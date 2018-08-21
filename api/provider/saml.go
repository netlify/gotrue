package provider

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/netlify/gotrue/conf"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"
	"golang.org/x/oauth2"
)

type SamlProvider struct {
	ServiceProvider *saml2.SAMLServiceProvider
}

func getMetadata(url string) (*types.EntityDescriptor, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
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
func NewSamlProvider(ext conf.SamlProviderConfiguration) (*SamlProvider, error) {
	if !ext.Enabled {
		return nil, errors.New("SAML Provider is not enabled")
	}

	if _, err := url.Parse(ext.MetadataURL); err != nil {
		return nil, fmt.Errorf("Metadata URL is invalid: %+v", err)
	}

	meta, err := getMetadata(ext.MetadataURL)
	if err != nil {
		return nil, err
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

	// TODO: generate keys once, save them in the database and use here
	randomKeyStore := dsig.RandomKeyStoreForTest()

	sp := &saml2.SAMLServiceProvider{
		IdentityProviderSSOURL:      ssoService.Location,
		IdentityProviderIssuer:      meta.EntityID,
		AssertionConsumerServiceURL: baseURI.String() + "/saml/acs",
		ServiceProviderIssuer:       baseURI.String() + "/saml",
		SignAuthnRequests:           true,
		AudienceURI:                 baseURI.String() + "/saml",
		IDPCertificateStore:         &certStore,
		SPKeyStore:                  randomKeyStore,
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

	rawMetadata, err := xml.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}
