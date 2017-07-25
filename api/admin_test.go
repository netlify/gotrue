package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AdminTestSuite struct {
	suite.Suite
	User *models.User
	API  *API
}

func (ts *AdminTestSuite) SetupTest() {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	require.NoError(ts.T(), err)
	ts.API = api
}

// TestAdminUsersUnauthorized tests API /admin/users route without authentication
func (ts *AdminTestSuite) TestAdminUsersUnauthorized() {
	// Setup request
	req := httptest.NewRequest("GET", "/admin/users", nil)

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	ts.API.adminUsers(ctx, w, req)

	assert.Equal(ts.T(), w.Code, 401)
}

func (ts *AdminTestSuite) makeSuperAdmin(req *http.Request, email string) (context.Context, *httptest.ResponseRecorder) {
	// Cleanup existing user, if they already exist
	if u, _ := ts.API.db.FindUserByEmailAndAudience(email, ts.API.config.JWT.Aud); u != nil {
		require.NoError(ts.T(), ts.API.db.DeleteUser(u), "Error deleting user")
	}

	u, err := models.NewUser(email, "test", ts.API.config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")

	u.IsSuperAdmin = true
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	token, err := ts.API.generateAccessToken(u)
	require.NoError(ts.T(), err, "Error generating access token")

	tok, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		assert.Equal(ts.T(), token.Header["alg"], "HS256")
		return []byte(ts.API.config.JWT.Secret), nil
	})
	require.NoError(ts.T(), err, "Error parsing token")

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	return withToken(ctx, tok), w
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers() {
	// Setup request
	req := httptest.NewRequest("GET", "/admin/users", nil)

	// Setup response recorder with super admin privileges
	ctx, w := ts.makeSuperAdmin(req, "test@example.com")

	ts.API.adminUsers(ctx, w, req)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), w.Code, 200)
	for _, user := range data["users"].([]interface{}) {
		assert.NotEmpty(ts.T(), user)
	}
}

// TestAdminUserCreate tests API /admin/user route (POST)
func (ts *AdminTestSuite) TestAdminUserCreate() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test1@example.com",
		"password": "test1",
	}))

	// Setup request
	req := httptest.NewRequest("POST", "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ctx, w := ts.makeSuperAdmin(req, "test@example.com")

	ts.API.adminUserCreate(ctx, w, req)

	assert.Equal(ts.T(), w.Code, 200)

	u, err := ts.API.db.FindUserByEmailAndAudience("test1@example.com", ts.API.config.JWT.Aud)
	require.NoError(ts.T(), err)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["email"], u.Email)
}

// TestAdminUserGet tests API /admin/user route (GET)
func (ts *AdminTestSuite) TestAdminUserGet() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"email": "test1@example.com",
			"aud":   ts.API.config.JWT.Aud,
		},
	}))

	u, err := models.NewUser("test1@example.com", "test", ts.API.config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	// Setup request
	req := httptest.NewRequest("GET", "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ctx, w := ts.makeSuperAdmin(req, "test@example.com")

	ts.API.adminUserGet(ctx, w, req)

	assert.Equal(ts.T(), w.Code, 200)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["email"], "test1@example.com")
}

// TestAdminUserUpdate tests API /admin/user route (UPDATE)
func (ts *AdminTestSuite) TestAdminUserUpdate() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"role": "testing",
		"user": map[string]interface{}{
			"email": "test1@example.com",
			"aud":   ts.API.config.JWT.Aud,
		},
	}))

	// Setup request
	req := httptest.NewRequest("UPDATE", "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ctx, w := ts.makeSuperAdmin(req, "test@example.com")

	ts.API.adminUserUpdate(ctx, w, req)

	assert.Equal(ts.T(), w.Code, 200)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["role"], "testing")

	u, err := ts.API.db.FindUserByEmailAndAudience("test1@example.com", ts.API.config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.Equal(ts.T(), u.Role, "testing")
}

// TestAdminUserDelete tests API /admin/user route (DELETE)
func (ts *AdminTestSuite) TestAdminUserDelete() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"email": "test1@example.com",
			"aud":   ts.API.config.JWT.Aud,
		},
	}))

	// Setup request
	req := httptest.NewRequest("DELETE", "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ctx, w := ts.makeSuperAdmin(req, "test@example.com")

	ts.API.adminUserDelete(ctx, w, req)

	assert.Equal(ts.T(), w.Code, 200)
}

func TestAdmin(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
