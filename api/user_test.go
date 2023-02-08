package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"context"
	"github.com/tigrisdata/tigris-client-go/tigris"
)

type UserTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	instanceID uuid.UUID
}

func TestUser(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &UserTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}

	suite.Run(t, ts)
}

func (ts *UserTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)

	// Create user
	u, err := models.NewUser(ts.instanceID, "test@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")

	_, err = tigris.GetCollection[models.User](ts.API.db).Insert(context.TODO(), u)
	require.NoError(ts.T(), err, "Error saving new test user")
}

func (ts *UserTestSuite) TestUser_UpdatePassword() {
	u, err := models.FindUserByEmailAndAudience(context.TODO(), ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"password": "newpass",
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPut, "http://localhost/user", &buffer)
	req.Header.Set("Content-Type", "application/json")
	tokenSigner := NewTokenSigner(ts.Config)

	token, err := generateAccessToken(u, time.Second*time.Duration(ts.Config.JWT.Exp), ts.Config, tokenSigner)
	require.NoError(ts.T(), err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Setup response recorder
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)

	u, err = models.FindUserByEmailAndAudience(req.Context(), ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	assert.True(ts.T(), u.Authenticate("newpass"))
}
