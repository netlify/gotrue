package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	jwt "github.com/golang-jwt/jwt/v4"
)

func (ts *ExternalTestSuite) TestSignupExternalGitlab() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=gitlab", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Gitlab.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Gitlab.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("read_user", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.API.config.OperatorToken), nil
	})
	ts.Require().NoError(err)

	ts.Equal("gitlab", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func GitlabTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string, emails string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Gitlab.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"gitlab_token","expires_in":100000}`)
		case "/api/v4/user":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		case "/api/v4/user/emails":
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, emails)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown gitlab oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Gitlab.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalGitlab_AuthorizationCode() {
	// additional emails from GitLab don't return confirm status
	ts.Config.Mailer.Autoconfirm = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"name":"GitLab Test","avatar_url":"http://example.com/avatar"}`
	emails := `[{"id":1,"email":"gitlab@example.com"}]`
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "gitlab", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "gitlab@example.com", "GitLab Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalGitLabDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"name":"Gitlab Test","avatar_url":"http://example.com/avatar"}`
	emails := `[{"id":1,"email":"gitlab@example.com"}]`
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "gitlab", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "gitlab@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalGitLabDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"name":"Gitlab Test","avatar_url":"http://example.com/avatar"}`
	emails := `[]`
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "gitlab", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "gitlab@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalGitLabDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("gitlab@example.com", "GitLab Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"gitlab@example.com","name":"GitLab Test","avatar_url":"http://example.com/avatar","confirmed_at":"2012-05-23T09:05:22Z"}`
	emails := "[]"
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "gitlab", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "gitlab@example.com", "GitLab Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalGitLabDisableSignupSuccessWithSecondaryEmail() {
	// additional emails from GitLab don't return confirm status
	ts.Config.Mailer.Autoconfirm = true
	ts.Config.DisableSignup = true

	ts.createUser("secondary@example.com", "GitLab Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"primary@example.com","name":"GitLab Test","avatar_url":"http://example.com/avatar"}`
	emails := `[{"id":1,"email":"secondary@example.com"}]`
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "gitlab", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "secondary@example.com", "GitLab Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitLabSuccessWhenMatchingToken() {
	// name and avatar should be populated from GitLab API
	ts.createUser("gitlab@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"gitlab@example.com","name":"GitLab Test","avatar_url":"http://example.com/avatar","confirmed_at":"2012-05-23T09:05:22Z"}`
	emails := "[]"
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "gitlab", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "gitlab@example.com", "GitLab Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitLabErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"gitlab@example.com","name":"GitLab Test","avatar_url":"http://example.com/avatar","confirmed_at":"2012-05-23T09:05:22Z"}`
	emails := "[]"
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "gitlab", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitLabErrorWhenWrongToken() {
	ts.createUser("gitlab@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"gitlab@example.com","name":"GitLab Test","avatar_url":"http://example.com/avatar","confirmed_at":"2012-05-23T09:05:22Z"}`
	emails := "[]"
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "gitlab", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitLabErrorWhenEmailDoesntMatch() {
	ts.createUser("gitlab@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"other@example.com","name":"GitLab Test","avatar_url":"http://example.com/avatar","confirmed_at":"2012-05-23T09:05:22Z"}`
	emails := "[]"
	server := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "gitlab", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}
