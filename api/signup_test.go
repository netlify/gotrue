package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SignupTestSuite struct {
	suite.Suite
	API *API
}

func (ts *SignupTestSuite) SetupTest() {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	require.NoError(ts.T(), err)

	ts.API = api

	// Cleanup existing user
	u, err := ts.API.db.FindUserByEmailAndAudience("test@example.com", api.config.JWT.Aud)
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
	req := httptest.NewRequest("POST", "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	ts.API.Signup(ctx, w, req)

	assert.Equal(ts.T(), w.Code, 200)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	assert.Equal(ts.T(), data["email"], "test@example.com")
	assert.Equal(ts.T(), data["aud"], ts.API.config.JWT.Aud)
	assert.Equal(ts.T(), data["user_metadata"].(map[string]interface{})["a"], 1.0)
	assert.Len(ts.T(), data, 12)
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
	req := httptest.NewRequest("POST", "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	y := httptest.NewRecorder()
	ctx := req.Context()

	ts.API.Signup(ctx, y, req)
	u, err := ts.API.db.FindUserByEmailAndAudience("test1@example.com", ts.API.config.JWT.Aud)
	if err == nil {
		u.Confirm()
		require.NoError(ts.T(), ts.API.db.UpdateUser(u))
	}

	encode()
	ts.API.Signup(ctx, w, req)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), w.Code, 500)
	assert.Equal(ts.T(), data["code"], 500.0)
}

func (ts *SignupTestSuite) TestVerifySignup() {

	user, err := models.NewUser("test@example.com", "testing", ts.API.config.JWT.Aud, nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), ts.API.db.CreateUser(user))

	// Find test user
	u, err := ts.API.db.FindUserByEmailAndAudience("test@example.com", ts.API.config.JWT.Aud)
	require.NoError(ts.T(), err)

	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "signup",
		"token": u.ConfirmationToken,
	}))

	// Setup request
	req := httptest.NewRequest("POST", "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	ts.API.Verify(ctx, w, req)

	assert.Equal(ts.T(), w.Code, 200)
}

func TestSignup(t *testing.T) {
	suite.Run(t, new(SignupTestSuite))
}
