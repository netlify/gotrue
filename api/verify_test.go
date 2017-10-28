package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type VerifyTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration
}

func (ts *VerifyTestSuite) SetupSuite() {
	require.NoError(ts.T(), os.Setenv("GOTRUE_DB_DATABASE_URL", createTestDB()))
}

func (ts *VerifyTestSuite) TearDownSuite() {
	os.Remove(ts.API.config.DB.URL)
}

func (ts *VerifyTestSuite) SetupTest() {
	api, config, err := NewAPIFromConfigFile("test.env", "v1")
	require.NoError(ts.T(), err)

	ts.API = api
	ts.Config = config

	// Cleanup existing user
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", config.JWT.Aud)
	if err == nil {
		require.NoError(ts.T(), api.db.DeleteUser(u))
	}

	// Create user
	u, err = models.NewUser("", "test@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	require.NoError(ts.T(), api.db.CreateUser(u), "Error saving new test user")
}

func TestVerify(t *testing.T) {
	suite.Run(t, new(VerifyTestSuite))
}

func (ts *VerifyTestSuite) TestVerify_PasswordRecovery() {
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &time.Time{}
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

	assert.WithinDuration(ts.T(), time.Now(), *u.RecoverySentAt, 1*time.Second)
	assert.False(ts.T(), u.IsConfirmed())

	// Send Verify request
	var vbuffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&vbuffer).Encode(map[string]interface{}{
		"type":  "recovery",
		"token": u.RecoveryToken,
	}))

	req = httptest.NewRequest(http.MethodPost, "http://localhost/verify", &vbuffer)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusOK)

	u, err = ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.True(ts.T(), u.IsConfirmed())
}
