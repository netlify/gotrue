package provider

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"

	"github.com/netlify/gotrue/conf"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/gobuffalo/uuid"
	"golang.org/x/oauth2"
)

type SamlProvider struct {
	ServiceProvider *saml2.SAMLServiceProvider
}

type ConfigX509KeyStore struct {
	InstanceID uuid.UUID
	DB         *storage.Connection
	Conf       conf.SamlProviderConfiguration
}

func getMetadata(url string) (*types.EntityDescriptor, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("Request failed with status %s", res.Status)
	}

	rawMetadata, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	metadata := &types.EntityDescriptor{}
	err = xml.Unmarshal(rawMetadata, metadata)
	if err != nil {
		return nil, err
	}

	// TODO: cache in memory

	return metadata, nil
}

// NewSamlProvider creates a Saml account provider.
func NewSamlProvider(ext conf.SamlProviderConfiguration, db *storage.Connection, instanceId uuid.UUID) (*SamlProvider, error) {
	if !ext.Enabled {
		return nil, errors.New("SAML Provider is not enabled")
	}

	if _, err := url.Parse(ext.MetadataURL); err != nil {
		return nil, fmt.Errorf("Metadata URL is invalid: %+v", err)
	}

	meta, err := getMetadata(ext.MetadataURL)
	if err != nil {
		return nil, fmt.Errorf("Fetching metadata failed: %+v", err)
	}

	baseURI, err := url.Parse(strings.Trim(ext.APIBase, "/"))
	if err != nil || ext.APIBase == "" {
		return nil, fmt.Errorf("Invalid API base URI: %s", ext.APIBase)
	}

	var ssoService types.SingleSignOnService
	foundService := false
	for _, service := range meta.IDPSSODescriptor.SingleSignOnServices {
		if service.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
			ssoService = service
			foundService = true
			break
		}
	}
	if !foundService {
		return nil, errors.New("No valid SSO service found in IDP metadata")
	}

	certStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}

	for _, kd := range meta.IDPSSODescriptor.KeyDescriptors {
		for _, xcert := range kd.KeyInfo.X509Data.X509Certificates {
			if xcert.Data == "" {
				continue
			}
			certData, err := base64.StdEncoding.DecodeString(xcert.Data)
			if err != nil {
				continue
			}

			idpCert, err := x509.ParseCertificate(certData)
			if err != nil {
				continue
			}

			certStore.Roots = append(certStore.Roots, idpCert)
		}
	}

	keyStore := &ConfigX509KeyStore{
		InstanceID: instanceId,
		DB:         db,
		Conf:       ext,
	}

	sp := &saml2.SAMLServiceProvider{
		IdentityProviderSSOURL:      ssoService.Location,
		IdentityProviderIssuer:      meta.EntityID,
		AssertionConsumerServiceURL: baseURI.String() + "/saml/acs",
		ServiceProviderIssuer:       baseURI.String() + "/saml",
		SignAuthnRequests:           true,
		AudienceURI:                 baseURI.String() + "/saml",
		IDPCertificateStore:         &certStore,
		SPKeyStore:                  keyStore,
		AllowMissingAttributes:      true,
	}

	p := &SamlProvider{
		ServiceProvider: sp,
	}
	return p, nil
}

func (p SamlProvider) AuthCodeURL(tokenString string, args ...oauth2.AuthCodeOption) string {
	url, err := p.ServiceProvider.BuildAuthURL(tokenString)
	if err != nil {
		return ""
	}
	return url
}

func (p SamlProvider) SPMetadata() ([]byte, error) {
	metadata, err := p.ServiceProvider.Metadata()
	if err != nil {
		return nil, err
	}

	// the typing for encryption methods currently causes the xml to violate the spec
	// therefore they are removed since they are optional anyways and mostly unused
	metadata.SPSSODescriptor.KeyDescriptors[1].EncryptionMethods = []types.EncryptionMethod{}

	rawMetadata, err := xml.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

func (ks ConfigX509KeyStore) GetKeyPair() (*rsa.PrivateKey, []byte, error) {
	if ks.Conf.SigningCert == "" && ks.Conf.SigningKey == "" {
		return ks.CreateSigningCert()
	}

	keyPair, err := tls.X509KeyPair([]byte(ks.Conf.SigningCert), []byte(ks.Conf.SigningKey))
	if err != nil {
		return nil, nil, fmt.Errorf("Parsing key pair failed: %+v", err)
	}

	var privKey *rsa.PrivateKey
	switch key := keyPair.PrivateKey.(type) {
	case *rsa.PrivateKey:
		privKey = key
	default:
		return nil, nil, errors.New("Private key is not an RSA key")
	}

	return privKey, keyPair.Certificate[0], nil
}

func (ks ConfigX509KeyStore) CreateSigningCert() (*rsa.PrivateKey, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	currentTime := time.Now()

	certBody := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    currentTime.Add(-5 * time.Minute),
		NotAfter:     currentTime.Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, certBody, certBody, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create certificate: %+v", err)
	}

	if err := ks.SaveConfig(cert, key); err != nil {
		return nil, nil, fmt.Errorf("Saving signing keypair failed: %+v", err)
	}

	return key, cert, nil
}

func (ks ConfigX509KeyStore) SaveConfig(cert []byte, key *rsa.PrivateKey) error {
	if ks.InstanceID == uuid.Nil {
		return nil
	}

	pemCert := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}

	certBytes := pem.EncodeToMemory(pemCert)
	if certBytes == nil {
		return errors.New("Could not encode certificate")
	}

	pemKey := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	keyBytes := pem.EncodeToMemory(pemKey)
	if keyBytes == nil {
		return errors.New("Could not encode key")
	}

	instance, err := models.GetInstance(ks.DB, ks.InstanceID)
	if err != nil {
		return err
	}

	conf := instance.BaseConfig
	conf.External.Saml.SigningCert = string(certBytes)
	conf.External.Saml.SigningKey = string(keyBytes)

	if err := instance.UpdateConfig(ks.DB, conf); err != nil {
		return err
	}

	return nil
}
