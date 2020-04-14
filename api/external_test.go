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

func BitbucketTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string, emails string) (*httptest.Server, *url.URL) {
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

	// get redirect url w/ state
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=bitbucket", nil)
	req.Header.Set("Referer", "https://example.netlify.com/admin")
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
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
	req = httptest.NewRequest(http.MethodGet, testURL.String(), nil)
	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err = url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	ts.Require().Equal("/admin", u.Path)

	return server, u
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucket_AuthorizationCode() {
	ts.Config.DisableSignup = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server, u := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "bitbucket@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("Bitbucket Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucketDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server, u := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

	// ensure new sign ups error
	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().Equal(v.Get("error_description"), "Signups not allowed for this instance")
	ts.Require().Equal(v.Get("error"), "access_denied")

	ts.Empty(v.Get("access_token"))
	ts.Empty(v.Get("refresh_token"))
	ts.Empty(v.Get("expires_in"))
	ts.Empty(v.Get("token_type"))

	// ensure user is nil
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "github@example.com", ts.Config.JWT.Aud)
	ts.Require().Error(err, "User not found")
	ts.Require().Nil(user)
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucketDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("bitbucket@example.com", "Bitbucket Test", "http://example.com/avatar")

	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"bitbucket@example.com","is_primary":true,"is_confirmed":true}]}`
	server, u := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "bitbucket@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("Bitbucket Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func (ts *ExternalTestSuite) TestSignupExternalBitbucketDisableSignupSuccessWithSecondaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("secondary@example.com", "Bitbucket Test", "http://example.com/avatar")

	tokenCount, userCount := 0, 0
	code := "authcode"
	bitbucketUser := `{"display_name":"Bitbucket Test","avatar":{"href":"http://example.com/avatar"}}`
	emails := `{"values":[{"email":"primary@example.com","is_primary":true,"is_confirmed":true},{"email":"secondary@example.com","is_primary":false,"is_confirmed":true}]}`
	server, u := BitbucketTestSignupSetup(ts, &tokenCount, &userCount, code, bitbucketUser, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "secondary@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("Bitbucket Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
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

func GitlabTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string, emails string) (*httptest.Server, *url.URL) {
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

	// get redirect url w/ state
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=gitlab", nil)
	req.Header.Set("Referer", "https://example.netlify.com/admin")
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
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
	req = httptest.NewRequest(http.MethodGet, testURL.String(), nil)
	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err = url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	ts.Require().Equal("/admin", u.Path)

	return server, u
}

func (ts *ExternalTestSuite) TestSignupExternalGitlab_AuthorizationCode() {
	ts.Config.Mailer.Autoconfirm = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"name":"Gitlab Test","avatar_url":"http://example.com/avatar"}`
	emails := `[{"id":1,"email":"gitlab@example.com"}]`
	server, u := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "gitlab@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("Gitlab Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func (ts *ExternalTestSuite) TestSignupExternalGitLabDisableSignupErrorWhenNoUser() {
	// additional emails from GitLab don't return confirm status
	ts.Config.Mailer.Autoconfirm = true
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"name":"Gitlab Test","avatar_url":"http://example.com/avatar"}`
	emails := `[{"id":1,"email":"gitlab@example.com"}]`
	server, u := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

	// ensure new sign ups error
	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().Equal(v.Get("error_description"), "Signups not allowed for this instance")
	ts.Require().Equal(v.Get("error"), "access_denied")

	ts.Empty(v.Get("access_token"))
	ts.Empty(v.Get("refresh_token"))
	ts.Empty(v.Get("expires_in"))
	ts.Empty(v.Get("token_type"))

	// ensure user is nil
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "github@example.com", ts.Config.JWT.Aud)
	ts.Require().Error(err, "User not found")
	ts.Require().Nil(user)
}

func (ts *ExternalTestSuite) TestSignupExternalGitLabDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("gitlab@example.com", "GitLab Test", "http://example.com/avatar")

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"gitlab@example.com","name":"Gitlab Test","avatar_url":"http://example.com/avatar","confirmed_at":"2012-05-23T09:05:22Z"}`
	emails := "[]"
	server, u := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "gitlab@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("GitLab Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func (ts *ExternalTestSuite) TestSignupExternalGitLabDisableSignupSuccessWithSecondaryEmail() {
	// additional emails from GitLab don't return confirm status
	ts.Config.Mailer.Autoconfirm = true
	ts.Config.DisableSignup = true

	ts.createUser("secondary@example.com", "GitLab Test", "http://example.com/avatar")

	tokenCount, userCount := 0, 0
	code := "authcode"
	gitlabUser := `{"email":"primary@example.com","name":"Gitlab Test","avatar_url":"http://example.com/avatar"}`
	emails := `[{"id":1,"email":"secondary@example.com"}]`
	server, u := GitlabTestSignupSetup(ts, &tokenCount, &userCount, code, gitlabUser, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "secondary@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("GitLab Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func GitHubTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, emails string) (*httptest.Server, *url.URL) {
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

	// get redirect url w/ state
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=github", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
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
	req = httptest.NewRequest(http.MethodGet, testURL.String(), nil)
	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err = url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")

	return server, u
}

func (ts *ExternalTestSuite) TestSignupExternalGitHub_AuthorizationCode() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server, u := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "github@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("GitHub Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func (ts *ExternalTestSuite) TestSignupExternalGitHubDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true
	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server, u := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

	// ensure new sign ups error
	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().Equal(v.Get("error_description"), "Signups not allowed for this instance")
	ts.Require().Equal(v.Get("error"), "access_denied")

	ts.Empty(v.Get("access_token"))
	ts.Empty(v.Get("refresh_token"))
	ts.Empty(v.Get("expires_in"))
	ts.Empty(v.Get("token_type"))

	// ensure user is nil
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "github@example.com", ts.Config.JWT.Aud)
	ts.Require().Error(err, "User not found")
	ts.Require().Nil(user)
}

func (ts *ExternalTestSuite) TestSignupExternalGitHubDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("github@example.com", "GitHub Test", "http://example.com/avatar")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"github@example.com", "primary": true, "verified": true}]`
	server, u := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "github@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("GitHub Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func (ts *ExternalTestSuite) TestSignupExternalGitHubDisableSignupSuccessWithNonPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("secondary@example.com", "GitHub Test", "http://example.com/avatar")

	tokenCount, userCount := 0, 0
	code := "authcode"
	emails := `[{"email":"primary@example.com", "primary": true, "verified": true},{"email":"secondary@example.com", "primary": false, "verified": true}]`
	server, u := GitHubTestSignupSetup(ts, &tokenCount, &userCount, code, emails)
	defer server.Close()

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
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "secondary@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("GitHub Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
}

func (ts *ExternalTestSuite) createUser(email string, name string, avatar string) (*models.User, error) {
	// Cleanup existing user, if they already exist
	if u, _ := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, email, ts.Config.JWT.Aud); u != nil {
		require.NoError(ts.T(), ts.API.db.Destroy(u), "Error deleting user")
	}

	u, err := models.NewUser(ts.instanceID, email, "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": name, "avatar_url": avatar})
	ts.Require().NoError(err, "Error making new user")
	ts.Require().NoError(ts.API.db.Create(u), "Error creating user")

	return u, err
}
