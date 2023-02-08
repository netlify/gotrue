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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"context"
)

type VerifyTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	instanceID uuid.UUID
}

func TestVerify(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &VerifyTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}

	suite.Run(t, ts)
}

func (ts *VerifyTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)

	// Create user
	u, err := models.NewUser(ts.instanceID, "test@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")

	_, err = tigris.GetCollection[models.User](ts.API.db).Insert(context.TODO(), u)
	require.NoError(ts.T(), err, "Error saving new test user")
}

func (ts *VerifyTestSuite) TestVerify_PasswordRecovery() {
	u, err := models.FindUserByEmailAndAudience(context.TODO(), ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &time.Time{}

	_, err = tigris.GetCollection[models.User](ts.API.db).InsertOrReplace(context.TODO(), u)
	require.NoError(ts.T(), err)

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
	assert.Equal(ts.T(), http.StatusOK, w.Code)

	u, err = models.FindUserByEmailAndAudience(req.Context(), ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
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
	assert.Equal(ts.T(), http.StatusOK, w.Code)

	u, err = models.FindUserByEmailAndAudience(req.Context(), ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.True(ts.T(), u.IsConfirmed())
}
