package api

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	api, config, err := NewAPIFromConfigFile("test.env", "v1")
	require.NoError(ts.T(), err)

	ts.API = api
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

	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := models.User{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	assert.Equal(ts.T(), "test@example.com", data.Email)
	assert.Equal(ts.T(), ts.Config.JWT.Aud, data.Aud)
	assert.Equal(ts.T(), 1.0, data.UserMetaData["a"])
	assert.Equal(ts.T(), "email", data.AppMetaData["provider"])
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

// TestSignupThrottle checks to make sure endpoint isn't called more than 1 time/second
func (ts *SignupTestSuite) TestSignupThrottle() {
	ts.API.config.Throttle.Enabled = true

	// Request body
	var buffer bytes.Buffer

	encode := func(userid string) {
		require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
			"email":    fmt.Sprintf("%s@example.com", userid),
			"password": userid,
		}))
	}

	encode("test2")

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	y := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(y, req)
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test2@example.com", ts.Config.JWT.Aud)
	if err == nil {
		u.Confirm()
		require.NoError(ts.T(), ts.API.db.UpdateUser(u))
	}

	encode("test3")
	ts.API.handler.ServeHTTP(w, req)

	u, err = ts.API.db.FindUserByEmailAndAudience("", "test3@example.com", ts.Config.JWT.Aud)
	require.Error(ts.T(), err, "expected user not to exist")

	assert.Equal(ts.T(), http.StatusTooManyRequests, w.Code)
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
