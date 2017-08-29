package api

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type UserTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration
}

func (ts *UserTestSuite) SetupSuite() {
	require.NoError(ts.T(), os.Setenv("GOTRUE_DB_DATABASE_URL", createTestDB()))
}

func (ts *UserTestSuite) TearDownSuite() {
	os.Remove(ts.API.config.DB.URL)
}

func (ts *UserTestSuite) SetupTest() {
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
	u, err = models.NewUser("", "test@example.com", "password", config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	require.NoError(ts.T(), api.db.CreateUser(u), "Error saving new test user")
}

func TestUser(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}

func (ts *UserTestSuite) TestUser_UpdatePassword() {
	u, err := ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"password": "newpass",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPut, "http://localhost/user", &buffer)
	req.Header.Set("Content-Type", "application/json")

	token, err := generateAccessToken(u, time.Second*time.Duration(ts.Config.JWT.Exp), ts.Config.JWT.Secret)
	require.NoError(ts.T(), err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Setup response recorder
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)

	u, err = ts.API.db.FindUserByEmailAndAudience("", "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	assert.True(ts.T(), u.Authenticate("newpass"))
}
