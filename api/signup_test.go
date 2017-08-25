package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SignupTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration
}

func (ts *SignupTestSuite) SetupSuite() {
	require.NoError(ts.T(), os.Setenv("GOTRUE_DB_DATABASE_URL", createTestDB()))
}

func (ts *SignupTestSuite) TearDownSuite() {
	os.Remove(ts.API.config.DB.URL)
}

func (ts *SignupTestSuite) SetupTest() {
	api, err := NewAPIFromConfigFile("test.env", "v1")
	require.NoError(ts.T(), err)

	ts.API = api

	config, err := conf.LoadConfig("test.env")
	require.NoError(ts.T(), err)
	ts.Config = config

	// Cleanup existing user
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", config.JWT.Aud)
	if err == nil {
		require.NoError(ts.T(), api.db.DeleteUser(u))
	}
}

// TestSignup tests API /signup route
func (ts *SignupTestSuite) TestSignup() {
	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test@example.com",
		"password": "test",
		"data": map[string]interface{}{
			"a": 1,
		},
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), w.Code, http.StatusOK)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	assert.Equal(ts.T(), data["email"], "test@example.com")
	assert.Equal(ts.T(), data["aud"], ts.Config.JWT.Aud)
	assert.Equal(ts.T(), data["user_metadata"].(map[string]interface{})["a"], 1.0)
	assert.Len(ts.T(), data, 13)
}

// TestSignupExternalUnsupported tests API /signup for an unsupported external provider
func (ts *SignupTestSuite) TestSignupExternalUnsupported() {
	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"provider": "external provider",
		"code":     "123456789",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)

	// Bad request expected for invalid external provider
	assert.Equal(ts.T(), w.Code, http.StatusBadRequest)
}

// TestSignupExternalGithub tests API /signup for github
func (ts *SignupTestSuite) TestSignupExternalGithub() {
	code := os.Getenv("GOTRUE_GITHUB_OAUTH_CODE")
	if code == "" || ts.Config.External.Github.Secret == "" {
		ts.T().Skip("GOTRUE_GITHUB_OAUTH_CODE or Github external provider config not set")
		return
	}

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"provider": "github",
		"code":     code,
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), w.Code, http.StatusOK)
}

// TestSignupExternalBitbucket tests API /signup for bitbucket
func (ts *SignupTestSuite) TestSignupExternalBitbucket() {
	code := os.Getenv("GOTRUE_BITBUCKET_OAUTH_CODE")
	if code == "" || ts.Config.External.Bitbucket.Secret == "" {
		ts.T().Skip("GOTRUE_BITBUCKET_OAUTH_CODE or Bitbucket external provider config not set")
		return
	}

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"provider": "bitbucket",
		"code":     code,
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), w.Code, http.StatusOK)
}

// TestSignupExternalGitlab tests API /signup for gitlab
func (ts *SignupTestSuite) TestSignupExternalGitlab() {
	code := os.Getenv("GOTRUE_GITLAB_OAUTH_CODE")
	if code == "" || ts.Config.External.Gitlab.Secret == "" {
		ts.T().Skip("GOTRUE_GITLAB_OAUTH_CODE or Gitlab external provider config not set")
		return
	}

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"provider": "gitlab",
		"code":     code,
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusOK)
}

// TestSignupExternalGoogle tests API /signup for google
func (ts *SignupTestSuite) TestSignupExternalGoogle() {
	code := os.Getenv("GOTRUE_GOOGLE_OAUTH_CODE")
	if code == "" || ts.Config.External.Google.Secret == "" {
		ts.T().Skip("GOTRUE_GOOGLE_OAUTH_CODE or Google external provider config not set")
		return
	}

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"provider": "google",
		"code":     code,
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusOK)
}

// TestSignupTwice checks to make sure the same email cannot be registered twice
func (ts *SignupTestSuite) TestSignupTwice() {
	// Request body
	var buffer bytes.Buffer

	encode := func() {
		require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
			"email":    "test1@example.com",
			"password": "test1",
			"data": map[string]interface{}{
				"a": 1,
			},
		}))
	}

	encode()

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	y := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(y, req)
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test1@example.com", ts.Config.JWT.Aud)
	if err == nil {
		u.Confirm()
		require.NoError(ts.T(), ts.API.db.UpdateUser(u))
	}

	encode()
	ts.API.handler.ServeHTTP(w, req)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), w.Code, http.StatusBadRequest)
	assert.Equal(ts.T(), data["code"], float64(http.StatusBadRequest))
}

func (ts *SignupTestSuite) TestVerifySignup() {

	user, err := models.NewUser("", "test@example.com", "testing", ts.Config.JWT.Aud, nil)
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

	assert.Equal(ts.T(), w.Code, http.StatusOK)
}

func TestSignup(t *testing.T) {
	suite.Run(t, new(SignupTestSuite))
}
