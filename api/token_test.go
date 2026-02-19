package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TokenTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	instanceID uuid.UUID
}

func TestToken(t *testing.T) {
	os.Setenv("GOTRUE_RATE_LIMIT_HEADER", "My-Custom-Header")
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &TokenTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *TokenTestSuite) SetupTest() {
	require.NoError(ts.T(), models.TruncateAll(ts.API.db))
}

// TestAccessTokenAudIsString verifies that the "aud" claim in generated JWTs is
// a JSON string, not an array. Git Gateway and other consumers expect a string
// and will fail to unmarshal an array.
func TestAccessTokenAudIsString(t *testing.T) {
	user := &models.User{Email: "test@example.com", Aud: "myapp"}
	user.ID = uuid.Must(uuid.NewV4())

	tokenStr, err := generateAccessToken(user, time.Hour, "test-secret")
	require.NoError(t, err)

	// Decode the payload (second segment) without signature validation
	parts := strings.Split(tokenStr, ".")
	require.Len(t, parts, 3, "JWT should have 3 segments")

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &raw))

	audRaw, ok := raw["aud"]
	require.True(t, ok, "aud claim should be present")
	assert.Equal(t, byte('"'), audRaw[0], "aud claim should be a JSON string, got: %s", string(audRaw))

	var aud string
	require.NoError(t, json.Unmarshal(audRaw, &aud), "aud claim should unmarshal as a string, got: %s", string(audRaw))
	assert.Equal(t, "myapp", aud)
}

func (ts *TokenTestSuite) TestRateLimitToken() {
	var buffer bytes.Buffer
	req := httptest.NewRequest(http.MethodPost, "http://localhost/token", &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("My-Custom-Header", "1.2.3.4")

	// It rate limits after 30 requests
	for i := 0; i < 30; i++ {
		w := httptest.NewRecorder()
		ts.API.handler.ServeHTTP(w, req)
		assert.Equal(ts.T(), http.StatusBadRequest, w.Code)
	}
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusTooManyRequests, w.Code)

	// It ignores X-Forwarded-For by default
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusTooManyRequests, w.Code)

	// It doesn't rate limit a new value for the limited header
	req = httptest.NewRequest(http.MethodPost, "http://localhost/token", &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("My-Custom-Header", "5.6.7.8")
	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)
}
