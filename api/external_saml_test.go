package api

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/gobuffalo/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExternalSamlTestSuite struct {
	suite.Suite
	API        *API
	Config     *conf.Configuration
	instanceID uuid.UUID
}

func TestExternalSaml(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &ExternalSamlTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *ExternalSamlTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)
}

func (ts *ExternalSamlTestSuite) docFromTemplate(path string, data interface{}) *etree.Document {
	doc := etree.NewDocument()

	templ, err := template.ParseFiles(path)
	ts.Require().NoError(err)
	read, write := io.Pipe()
	go func() {
		defer write.Close()
		err := templ.Execute(write, data)
		ts.Require().NoError(err)
	}()
	_, err = doc.ReadFrom(read)
	ts.Require().NoError(err)

	return doc
}

func (ts *ExternalSamlTestSuite) setupSamlExampleResponse(keyStore dsig.X509KeyStore) string {
	path := filepath.Join("testdata", "saml-response.xml")
	type ResponseParams struct {
		Now       string
		NotBefore string
		NotAfter  string
	}
	now := time.Now()
	doc := ts.docFromTemplate(path, ResponseParams{
		Now:       now.Format(time.RFC3339),
		NotBefore: now.Add(-5 * time.Minute).Format(time.RFC3339),
		NotAfter:  now.Add(5 * time.Minute).Format(time.RFC3339),
	})

	// sign
	resp := doc.SelectElement("Response")
	ctx := dsig.NewDefaultSigningContext(keyStore)
	sig, err := ctx.ConstructSignature(resp, true)
	ts.Require().NoError(err, "Response signature failed")
	respWithSig := resp.Copy()
	var children []etree.Token
	children = append(children, respWithSig.Child[0])     // issuer is always first
	children = append(children, sig)                      // next is the signature
	children = append(children, respWithSig.Child[1:]...) // then all other children
	respWithSig.Child = children
	doc.SetRoot(respWithSig)

	docRaw, err := doc.WriteToBytes()
	ts.Require().NoError(err)

	return base64.StdEncoding.EncodeToString(docRaw)
}

func (ts *ExternalSamlTestSuite) setupSamlExampleState() string {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=saml", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)

	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")

	urlBase, _ := url.Parse(u.String())
	urlBase.RawQuery = ""
	ts.Equal(urlBase.String(), "https://idp/saml2test/redirect")

	q := u.Query()
	state := q.Get("RelayState")
	ts.Require().NotEmpty(state)
	return state
}

func (ts *ExternalSamlTestSuite) setupSamlMetadata() (*httptest.Server, dsig.X509KeyStore) {
	idpKeyStore := dsig.RandomKeyStoreForTest()
	_, idpCert, _ := idpKeyStore.GetKeyPair()

	path := filepath.Join("testdata", "saml-idp-metadata.xml")
	type MetadataParams struct {
		Cert string
	}
	doc := ts.docFromTemplate(path, MetadataParams{Cert: base64.StdEncoding.EncodeToString(idpCert)})
	metadata, err := doc.WriteToString()
	ts.Require().NoError(err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(200)
		io.WriteString(w, metadata)
	}))
	return server, idpKeyStore
}

func (ts *ExternalSamlTestSuite) setupSamlSPCert() (string, string) {
	spKeyStore := dsig.RandomKeyStoreForTest()
	key, cert, _ := spKeyStore.GetKeyPair()
	keyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	certBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	return string(keyBytes), string(certBytes)
}

func (ts *ExternalSamlTestSuite) TestSignupExternalSaml_Callback() {
	server, idpKeyStore := ts.setupSamlMetadata()
	defer server.Close()
	ts.Config.External.Saml.MetadataURL = server.URL

	key, cert := ts.setupSamlSPCert()
	ts.Config.External.Saml.SigningKey = key
	ts.Config.External.Saml.SigningCert = cert

	form := url.Values{}
	form.Add("RelayState", ts.setupSamlExampleState())
	form.Add("SAMLResponse", ts.setupSamlExampleResponse(idpKeyStore))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/saml/acs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	ts.Require().Equal(http.StatusFound, w.Code)

	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")

	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Empty(v.Get("error_description"))
	ts.Empty(v.Get("error"))

	ts.NotEmpty(v.Get("access_token"))
	ts.NotEmpty(v.Get("refresh_token"))
	ts.NotEmpty(v.Get("expires_in"))
	ts.Equal("bearer", v.Get("token_type"))

	// ensure user has been created
	_, err = models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "saml@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
}

func (ts *ExternalSamlTestSuite) TestMetadata() {
	server, _ := ts.setupSamlMetadata()
	defer server.Close()
	ts.Config.External.Saml.MetadataURL = server.URL

	key, cert := ts.setupSamlSPCert()
	ts.Config.External.Saml.SigningKey = key
	ts.Config.External.Saml.SigningCert = cert

	req := httptest.NewRequest(http.MethodGet, "http://localhost/saml/metadata", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	ts.Require().Equal(http.StatusOK, w.Code)

	md := &types.EntityDescriptor{}
	err := xml.NewDecoder(w.Body).Decode(md)
	ts.Require().NoError(err)

	ts.Equal("http://localhost/saml", md.EntityID)
	for _, acs := range md.SPSSODescriptor.AssertionConsumerServices {
		if acs.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" {
			ts.Equal("http://localhost/saml/acs", acs.Location)
			break
		}
	}
}
