package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExternalTestSuite struct {
	suite.Suite
	API        *API
	Config     *conf.Configuration
	instanceID uuid.UUID
}

func TestExternal(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &ExternalTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *ExternalTestSuite) SetupTest() {
	ts.Config.DisableSignup = false
	ts.Config.Mailer.Autoconfirm = false

	models.TruncateAll(ts.API.db)
}

func (ts *ExternalTestSuite) createUser(email string, name string, avatar string, confirmationToken string) (*models.User, error) {
	// Cleanup existing user, if they already exist
	if u, _ := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, email, ts.Config.JWT.Aud); u != nil {
		require.NoError(ts.T(), ts.API.db.Destroy(u), "Error deleting user")
	}

	u, err := models.NewUser(ts.instanceID, email, "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": name, "avatar_url": avatar})

	if confirmationToken != "" {
		u.ConfirmationToken = confirmationToken
	}
	ts.Require().NoError(err, "Error making new user")
	ts.Require().NoError(ts.API.db.Create(u), "Error creating user")

	return u, err
}

func performAuthorizationRequest(ts *ExternalTestSuite, provider string, inviteToken string) *httptest.ResponseRecorder {
	authorizeURL := "http://localhost/authorize?provider=" + provider
	if inviteToken != "" {
		authorizeURL = authorizeURL + "&invite_token=" + inviteToken
	}

	req := httptest.NewRequest(http.MethodGet, authorizeURL, nil)
	req.Header.Set("Referer", "https://example.netlify.com/admin")
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	return w
}

func performAuthorization(ts *ExternalTestSuite, provider string, code string, inviteToken string) *url.URL {
	w := performAuthorizationRequest(ts, provider, inviteToken)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	state := q.Get("state")

	// auth server callback
	testURL, err := url.Parse("http://localhost/callback")
	ts.Require().NoError(err)
	v := testURL.Query()
	v.Set("code", code)
	v.Set("state", state)
	testURL.RawQuery = v.Encode()
	req := httptest.NewRequest(http.MethodGet, testURL.String(), nil)
	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err = url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	ts.Require().Equal("/admin", u.Path)

	return u
}

func assertAuthorizationSuccess(ts *ExternalTestSuite, u *url.URL, tokenCount int, userCount int, email string, name string, avatar string) {
	// ensure redirect has #access_token=...
	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().Empty(v.Get("error_description"))
	ts.Require().Empty(v.Get("error"))

	ts.NotEmpty(v.Get("access_token"))
	ts.NotEmpty(v.Get("refresh_token"))
	ts.NotEmpty(v.Get("expires_in"))
	ts.Equal("bearer", v.Get("token_type"))

	ts.Equal(1, tokenCount)
	ts.Equal(1, userCount)

	// ensure user has been created with metadata
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, email, ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal(name, user.UserMetaData["full_name"])
	ts.Equal(avatar, user.UserMetaData["avatar_url"])
}

func assertAuthorizationFailure(ts *ExternalTestSuite, u *url.URL, errorDescription string, errorType string, email string) {
	// ensure new sign ups error
	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().Equal(errorDescription, v.Get("error_description"))
	ts.Require().Equal(errorType, v.Get("error"))

	ts.Empty(v.Get("access_token"))
	ts.Empty(v.Get("refresh_token"))
	ts.Empty(v.Get("expires_in"))
	ts.Empty(v.Get("token_type"))

	// ensure user is nil
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, email, ts.Config.JWT.Aud)
	ts.Require().Error(err, "User not found")
	ts.Require().Nil(user)
}

// TestSignupExternalUnsupported tests API /authorize for an unsupported external provider
func (ts *ExternalTestSuite) TestSignupExternalUnsupported() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=external", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Equal(w.Code, http.StatusBadRequest)
}

// TestSignupExternalGithub tests API /authorize for github
func (ts *ExternalTestSuite) TestSignupExternalGithub() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=github", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Github.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Github.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("user:email", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.API.config.OperatorToken), nil
	})
	ts.Require().NoError(err)

	ts.Equal("github", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func GitHubTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, emails string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Github.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"github_token","expires_in":100000}`)
		case "/api/v3/user":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"GitHub Test","avatar_url":"http://example.com/avatar"}`)
		case "/api/v3/user/emails":
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, emails)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown github oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Github.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalGitHub_AuthorizationCode() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "github", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "github@example.com", "GitHub Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalGitHubDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "github", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "github@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalGitHubDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "github", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "github@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalGitHubDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("github@example.com", "GitHub Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "github", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "github@example.com", "GitHub Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalGitHubDisableSignupSuccessWithNonPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("secondary@example.com", "GitHub Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"primary@example.com", "primary": true, "verified": true},{"email":"secondary@example.com", "primary": false, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "github", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "secondary@example.com", "GitHub Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitHubSuccessWhenMatchingToken() {
	// name and avatar should be populated from GitHub API
	ts.createUser("github@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "github", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "github@example.com", "GitHub Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitHubErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "github", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitHubErrorWhenWrongToken() {
	ts.createUser("github@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "github", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGitHubErrorWhenEmailDoesntMatch() {
	ts.createUser("github@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"other@example.com", "primary": true, "verified": true}]`
	server := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	u := performAuthorization(ts, "github", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}

// TestSignupExternalBitbucket tests API /authorize for bitbucket
func (ts *ExternalTestSuite) TestSignupExternalBitbucket() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=bitbucket", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Bitbucket.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Bitbucket.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("account email", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.API.config.OperatorToken), nil
	})
	ts.Require().NoError(err)

	ts.Equal("bitbucket", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func BitbucketTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string, emails string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/site/oauth2/access_token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Bitbucket.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"bitbucket_token","expires_in":100000}`)
		case "/2.0/user":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		case "/2.0/user/emails":
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, emails)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown bitbucket oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Bitbucket.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucket_AuthorizationCode() {
	ts.Config.DisableSignup = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "bitbucket", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "bitbucket@example.com", "Bitbucket Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucketDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "bitbucket", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "bitbucket@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucketDisableSignupErrorWhenNoEmail() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "bitbucket", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "bitbucket@example.com")

}

func (ts *ExternalTestSuite) TestSignupExternalBitbucketDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("bitbucket@example.com", "Bitbucket Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "bitbucket", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "bitbucket@example.com", "Bitbucket Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucketDisableSignupSuccessWithSecondaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("secondary@example.com", "Bitbucket Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"primary@example.com","is_primary":true,"is_confirmed":true},{"email":"secondary@example.com","is_primary":false,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "bitbucket", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "secondary@example.com", "Bitbucket Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBitbucketSuccessWhenMatchingToken() {
	// name and avatar should be populated from Bitbucket API
	ts.createUser("bitbucket@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "bitbucket", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "bitbucket@example.com", "Bitbucket Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBitbucketErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "bitbucket", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBitbucketErrorWhenWrongToken() {
	ts.createUser("bitbucket@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	w := performAuthorizationRequest(ts, "bitbucket", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalBitbucketErrorWhenEmailDoesntMatch() {
	ts.createUser("bitbucket@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"other@example.com","is_primary":true,"is_confirmed":true}]}`
	server := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	u := performAuthorization(ts, "bitbucket", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}

// TestSignupExternalGitlab tests API /authorize for gitlab
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

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "github@example.com")
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

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "github@example.com")
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

// TestSignupExternalGoogle tests API /authorize for google
func (ts *ExternalTestSuite) TestSignupExternalGoogle() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=google", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Google.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Google.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("email profile", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.API.config.OperatorToken), nil
	})
	ts.Require().NoError(err)

	ts.Equal("google", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func GoogleTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/o/oauth2/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Google.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"google_token","expires_in":100000}`)
		case "/userinfo/v2/me":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown google oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Google.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalGoogle_AuthorizationCode() {
	ts.Config.DisableSignup = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","email":"google@example.com","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	u := performAuthorization(ts, "google", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "google@example.com", "Google Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalGoogleDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","email":"google@example.com","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	u := performAuthorization(ts, "google", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "google@example.com")
}
func (ts *ExternalTestSuite) TestSignupExternalGoogleDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	u := performAuthorization(ts, "google", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "google@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalGoogleDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("google@example.com", "Google Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","email":"google@example.com","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	u := performAuthorization(ts, "google", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "google@example.com", "Google Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGoogleSuccessWhenMatchingToken() {
	// name and avatar should be populated from Google API
	ts.createUser("google@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","email":"google@example.com","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	u := performAuthorization(ts, "google", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "google@example.com", "Google Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGoogleErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","email":"google@example.com","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "google", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGoogleErrorWhenWrongToken() {
	ts.createUser("google@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","email":"google@example.com","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "google", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalGoogleErrorWhenEmailDoesntMatch() {
	ts.createUser("google@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	googleUser := `{"name":"Google Test","picture":"http://example.com/avatar","email":"other@example.com","verified_email":true}}`
	server := GoogleTestSignupSetup(ts, &tokenCount, &userCount, code, googleUser)
	defer server.Close()

	u := performAuthorization(ts, "google", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}

func (ts *ExternalTestSuite) TestSignupExternalFacebook() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=facebook", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Facebook.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Facebook.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("email", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.API.config.OperatorToken), nil
	})
	ts.Require().NoError(err)

	ts.Equal("facebook", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func FacebookTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/access_token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Facebook.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"facebook_token","expires_in":100000}`)
		case "/me":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown facebook oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Facebook.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalFacebook_AuthorizationCode() {
	ts.Config.DisableSignup = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","email":"facebook@example.com","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	u := performAuthorization(ts, "facebook", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "facebook@example.com", "Facebook Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestSignupExternalFacebookDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","email":"facebook@example.com","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	u := performAuthorization(ts, "facebook", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "facebook@example.com")
}
func (ts *ExternalTestSuite) TestSignupExternalFacebookDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	u := performAuthorization(ts, "facebook", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "facebook@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalFacebookDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("facebook@example.com", "Facebook Test", "http://example.com/avatar", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","email":"facebook@example.com","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	u := performAuthorization(ts, "facebook", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "facebook@example.com", "Facebook Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalFacebookSuccessWhenMatchingToken() {
	// name and avatar should be populated from Facebook API
	ts.createUser("facebook@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","email":"facebook@example.com","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	u := performAuthorization(ts, "facebook", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "facebook@example.com", "Facebook Test", "http://example.com/avatar")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalFacebookErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","email":"facebook@example.com","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "facebook", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalFacebookErrorWhenWrongToken() {
	ts.createUser("facebook@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","email":"facebook@example.com","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "facebook", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalFacebookErrorWhenEmailDoesntMatch() {
	ts.createUser("facebook@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	facebookUser := `{"name":"Facebook Test","first_name":"Facebook","last_name":"Test","email":"other@example.com","picture":{"data":{"url":"http://example.com/avatar"}}}}`
	server := FacebookTestSignupSetup(ts, &tokenCount, &userCount, code, facebookUser)
	defer server.Close()

	u := performAuthorization(ts, "facebook", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}
