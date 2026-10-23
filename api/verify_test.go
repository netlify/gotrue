package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
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
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *VerifyTestSuite) SetupTest() {
	require.NoError(ts.T(), models.TruncateAll(ts.API.db))

	// Create user
	u, err := models.NewUser(ts.instanceID, "test@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error saving new test user")
}

func (ts *VerifyTestSuite) TestVerify_PasswordRecovery() {
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

	u, err = models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.True(ts.T(), u.IsConfirmed())
}

func (ts *VerifyTestSuite) TestVerify_Confirmation_Expired() {
	u, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	u.ConfirmationToken = "expired-confirmation-token"
	expired := time.Now().Add(-48 * time.Hour)
	u.ConfirmationSentAt = &expired
	require.NoError(ts.T(), ts.API.db.Update(u))

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "signup",
		"token": u.ConfirmationToken,
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code)
}

func (ts *VerifyTestSuite) TestVerify_WhitespaceToken_Signup() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "signup",
		"token": " ",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code)
}

func (ts *VerifyTestSuite) TestVerify_WhitespaceToken_Recovery() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "recovery",
		"token": " ",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code)
}

func (ts *VerifyTestSuite) TestVerify_ZeroWidthToken() {
	for _, typ := range []string{"signup", "recovery"} {
		for _, token := range []string{"\u200b", "\u200c", "\u200d", "\u2060", "\ufeff", "\u180e", " \u200b "} {
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
				"type":  typ,
				"token": token,
			}))

			req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			assert.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code, "type=%s token=%q", typ, token)
		}
	}
}

func (ts *VerifyTestSuite) TestVerify_TokenFromOtherInstance() {
	other, err := models.NewUser(uuid.Must(uuid.NewV4()), "other@example.com", "password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err)
	other.ConfirmationToken = "other-confirmation-token"
	other.RecoveryToken = "other-recovery-token"
	require.NoError(ts.T(), ts.API.db.Create(other))

	for typ, token := range map[string]string{"signup": other.ConfirmationToken, "recovery": other.RecoveryToken} {
		var buffer bytes.Buffer
		require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
			"type":  typ,
			"token": token,
		}))

		req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		ts.API.handler.ServeHTTP(w, req)
		assert.Equal(ts.T(), http.StatusNotFound, w.Code, "type=%s", typ)
	}
}

func (ts *VerifyTestSuite) TestVerify_TabToken() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "signup",
		"token": "\t",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code)
}

func (ts *VerifyTestSuite) TestVerify_MultipleSpacesToken() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "signup",
		"token": "   ",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code)
}

func (ts *VerifyTestSuite) TestVerify_PasswordRecovery_Expired() {
	u, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	// Set a recovery token that was sent 48 hours ago (beyond 24h default max age)
	u.RecoveryToken = "expired-test-token"
	expired := time.Now().Add(-48 * time.Hour)
	u.RecoverySentAt = &expired
	require.NoError(ts.T(), ts.API.db.Update(u))

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "recovery",
		"token": u.RecoveryToken,
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusUnprocessableEntity, w.Code)
}
