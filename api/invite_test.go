package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type InviteTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	token string
}

func (ts *InviteTestSuite) TearDownTest() {
	os.Remove(ts.API.config.DB.URL)
}

func (ts *InviteTestSuite) SetupTest() {
	require.NoError(ts.T(), os.Setenv("GOTRUE_DB_DATABASE_URL", createTestDB()))
	api, config, err := NewAPIFromConfigFile("test.env", "v1")
	require.NoError(ts.T(), err)

	ts.API = api
	ts.Config = config

	// Cleanup existing user
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", config.JWT.Aud)
	if err == nil {
		require.NoError(ts.T(), api.db.DeleteUser(u))
	}

	// Setup response recorder with super admin privileges
	ts.token = ts.makeSuperAdmin("admin@example.com")
}

func (ts *InviteTestSuite) makeSuperAdmin(email string) string {
	// Cleanup existing user, if they already exist
	if u, _ := ts.API.db.FindUserByEmailAndAudience("", email, ts.Config.JWT.Aud); u != nil {
		require.NoError(ts.T(), ts.API.db.DeleteUser(u), "Error deleting user")
	}

	u, err := models.NewUser("", email, "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": "Test User"})
	require.NoError(ts.T(), err, "Error making new user")

	u.IsSuperAdmin = true
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	token, err := generateAccessToken(u, time.Second*time.Duration(ts.Config.JWT.Exp), ts.Config.JWT.Secret)
	require.NoError(ts.T(), err, "Error generating access token")

	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	require.NoError(ts.T(), err, "Error parsing token")

	return token
}

func (ts *InviteTestSuite) TestInvite() {
	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
		"data": map[string]interface{}{
			"a": 1,
		},
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/invite", &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusOK)
}

func (ts *InviteTestSuite) TestInvite_WithoutAccess() {
	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
		"data": map[string]interface{}{
			"a": 1,
		},
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/invite", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusUnauthorized)
}

func (ts *InviteTestSuite) TestVerifyInvite() {
	user, err := models.NewUser("", "test@example.com", "", ts.Config.JWT.Aud, nil)
	now := time.Now()
	user.InvitedAt = &now
	user.EncryptedPassword = ""
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.API.db.CreateUser(user))

	// Find test user
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":     "signup",
		"token":    u.ConfirmationToken,
		"password": "testing",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), w.Code, http.StatusOK)
}

func (ts *InviteTestSuite) TestVerifyInvite_NoPassword() {
	user, err := models.NewUser("", "test@example.com", "", ts.Config.JWT.Aud, nil)
	now := time.Now()
	user.InvitedAt = &now
	user.EncryptedPassword = ""
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.API.db.CreateUser(user))

	// Find test user
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "signup",
		"token": u.ConfirmationToken,
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), w.Code, http.StatusUnprocessableEntity)
}

func (ts *InviteTestSuite) TestInviteExternalGitlab() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Gitlab.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"gitlab_token","expires_in":100000}`)
		case "/api/v4/user":
			userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"Gitlab Test","avatar_url":"http://example.com/avatar"}`)
		case "/api/v4/user/emails":
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `[{"email":"gitlab@example.com"}]`)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown gitlab oauth call %s", r.URL.Path)
		}
	}))
	defer server.Close()
	ts.Config.External.Gitlab.URL = server.URL

	// invite user
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(InviteParams{
		Email: "gitlab@example.com",
	}))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/invite", &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusOK, w.Code)

	// Find test user
	user, err := ts.API.db.FindUserByEmailAndAudience("", "gitlab@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// get redirect url w/ state
	req = httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=gitlab&invite_token="+user.ConfirmationToken, nil)
	w = httptest.NewRecorder()
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

	// ensure redirect has #access_token=...
	v, err = url.ParseQuery(u.Fragment)
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
	user, err = ts.API.db.FindUserByEmailAndAudience("", "gitlab@example.com", ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal("Gitlab Test", user.UserMetaData["full_name"])
	ts.Equal("http://example.com/avatar", user.UserMetaData["avatar_url"])
	ts.Equal("gitlab", user.AppMetaData["provider"])
}

func (ts *InviteTestSuite) TestInviteExternalGitlab_MismatchedEmails() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Gitlab.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"gitlab_token","expires_in":100000}`)
		case "/api/v4/user":
			userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"Gitlab Test","avatar_url":"http://example.com/avatar"}`)
		case "/api/v4/user/emails":
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `[{"email":"gitlab+mismatch@example.com"}]`)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown gitlab oauth call %s", r.URL.Path)
		}
	}))
	defer server.Close()
	ts.Config.External.Gitlab.URL = server.URL

	// invite user
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(InviteParams{
		Email: "gitlab@example.com",
	}))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/invite", &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusOK, w.Code)

	// Find test user
	user, err := ts.API.db.FindUserByEmailAndAudience("", "gitlab@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// get redirect url w/ state
	req = httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=gitlab&invite_token="+user.ConfirmationToken, nil)
	w = httptest.NewRecorder()
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

	// ensure redirect has #access_token=...
	v, err = url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().NotEmpty(v.Get("error_description"))
	ts.Require().Equal("invalid_request", v.Get("error"))
}

func TestInvite(t *testing.T) {
	suite.Run(t, new(InviteTestSuite))
}
