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
	"github.com/gobuffalo/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RecoverTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	instanceID uuid.UUID
}

func TestRecover(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &RecoverTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *RecoverTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)

	// Create user
	u, err := models.NewUser(ts.instanceID, "test@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error saving new test user")
}

func (ts *RecoverTestSuite) TestRecover_FirstRecovery() {
	u, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &time.Time{}
	require.NoError(ts.T(), ts.API.db.Update(u))

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

	u, err = models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	assert.WithinDuration(ts.T(), time.Now(), *u.RecoverySentAt, 1*time.Second)
}

func (ts *RecoverTestSuite) TestRecover_NoEmailSent() {
	recoveryTime := time.Now().UTC().Add(-5 * time.Minute)
	u, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &recoveryTime
	require.NoError(ts.T(), ts.API.db.Update(u))

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

	u, err = models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// ensure it did not send a new email
	u1 := recoveryTime.Round(time.Second).Unix()
	u2 := u.RecoverySentAt.Round(time.Second).Unix()
	assert.Equal(ts.T(), u1, u2)
}

func (ts *RecoverTestSuite) TestRecover_NewEmailSent() {
	recoveryTime := time.Now().UTC().Add(-20 * time.Minute)
	u, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	u.RecoverySentAt = &recoveryTime
	require.NoError(ts.T(), ts.API.db.Update(u))

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

	u, err = models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// ensure it sent a new email
	assert.WithinDuration(ts.T(), time.Now(), *u.RecoverySentAt, 1*time.Second)
}
