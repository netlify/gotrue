package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RecoverTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration
}

func (ts *RecoverTestSuite) SetupTest() {
	api, err := NewAPIFromConfigFile("test.env", "v1")
	require.NoError(ts.T(), err)

	ts.API = api

	config, err := conf.LoadConfig("test.env")
	require.NoError(ts.T(), err)
	ts.Config = config

	// Create user
	u, err := models.NewUser("", "test@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	require.NoError(ts.T(), api.db.CreateUser(u), "Error saving new test user")
}

func TestRecover(t *testing.T) {
	suite.Run(t, new(RecoverTestSuite))
}

func (ts *RecoverTestSuite) TestRecover_FirstRecovery() {
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = time.Time{}
	require.NoError(ts.T(), ts.API.db.UpdateUser(u))

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/recover", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusOK)

	u, err = ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	assert.WithinDuration(ts.T(), time.Now(), u.RecoverySentAt, 1*time.Second)
}

func (ts *RecoverTestSuite) TestRecover_NoEmailSent() {
	recoveryTime := time.Now().UTC().Add(-5 * time.Minute)
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = recoveryTime
	require.NoError(ts.T(), ts.API.db.UpdateUser(u))

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/recover", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusOK)

	u, err = ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// ensure it did not send a new email
	assert.Equal(ts.T(), recoveryTime, u.RecoverySentAt)
}

func (ts *RecoverTestSuite) TestRecover_NewEmailSent() {
	recoveryTime := time.Now().UTC().Add(-20 * time.Minute)
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = recoveryTime
	require.NoError(ts.T(), ts.API.db.UpdateUser(u))

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email": "test@example.com",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/recover", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusOK)

	u, err = ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// ensure it sent a new email
	assert.WithinDuration(ts.T(), time.Now(), u.RecoverySentAt, 1*time.Second)
}
