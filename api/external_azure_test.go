package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	jwt "github.com/golang-jwt/jwt"
)

const (
	azureUser        string = `{"name":"Azure Test","email":"azure@example.com","sub":"azuretestid"}`
	azureUserNoEmail string = `{"name":"Azure Test","sub":"azuretestid"}`
)

func (ts *ExternalTestSuite) TestSignupExternalAzure() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=azure", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Azure.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Azure.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("openid", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	ts.Require().NoError(err)

	ts.Equal("azure", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func AzureTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/common/oauth2/v2.0/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Azure.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"azure_token","expires_in":100000}`)
		case "/oidc/userinfo":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown azure oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Azure.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalAzure_AuthorizationCode() {
	ts.Config.DisableSignup = false
	ts.createUser("azuretestid", "azure@example.com", "Azure Test", "", "")
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUser)
	defer server.Close()

	u := performAuthorization(ts, "azure", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "azure@example.com", "Azure Test", "azuretestid", "")
}

func (ts *ExternalTestSuite) TestSignupExternalAzureDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUser)
	defer server.Close()

	u := performAuthorization(ts, "azure", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "azure@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalAzureDisableSignupErrorWhenNoEmail() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUserNoEmail)
	defer server.Close()

	u := performAuthorization(ts, "azure", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "azure@example.com")

}

func (ts *ExternalTestSuite) TestSignupExternalAzureDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("azuretestid", "azure@example.com", "Azure Test", "", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUser)
	defer server.Close()

	u := performAuthorization(ts, "azure", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "azure@example.com", "Azure Test", "azuretestid", "")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalAzureSuccessWhenMatchingToken() {
	// name and avatar should be populated from Azure API
	ts.createUser("azuretestid", "azure@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUser)
	defer server.Close()

	u := performAuthorization(ts, "azure", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "azure@example.com", "Azure Test", "azuretestid", "")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalAzureErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	azureUser := `{"name":"Azure Test","avatar":{"href":"http://example.com/avatar"}}`
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "azure", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalAzureErrorWhenWrongToken() {
	ts.createUser("azuretestid", "azure@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	azureUser := `{"name":"Azure Test","avatar":{"href":"http://example.com/avatar"}}`
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "azure", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalAzureErrorWhenEmailDoesntMatch() {
	ts.createUser("azuretestid", "azure@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	azureUser := `{"name":"Azure Test", "email":"other@example.com", "avatar":{"href":"http://example.com/avatar"}}`
	server := AzureTestSignupSetup(ts, &tokenCount, &userCount, code, azureUser)
	defer server.Close()

	u := performAuthorization(ts, "azure", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}
