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

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AdminTestSuite struct {
	suite.Suite
	User   *models.User
	API    *API
	Config *conf.Configuration
}

func (ts *AdminTestSuite) SetupSuite() {
	require.NoError(ts.T(), os.Setenv("GOTRUE_DB_DATABASE_URL", createTestDB()))
}

func (ts *AdminTestSuite) TearDownSuite() {
	os.Remove(ts.API.config.DB.URL)
}

func (ts *AdminTestSuite) SetupTest() {
	api, config, err := NewAPIFromConfigFile("test.env", "v1")
	require.NoError(ts.T(), err)
	ts.API = api
	ts.Config = config
}

// TestAdminUsersUnauthorized tests API /admin/users route without authentication
func (ts *AdminTestSuite) TestAdminUsersUnauthorized() {
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusUnauthorized)
}

func (ts *AdminTestSuite) makeSuperAdmin(req *http.Request, email string) {
	// Cleanup existing user, if they already exist
	if u, _ := ts.API.db.FindUserByEmailAndAudience("", email, ts.Config.JWT.Aud); u != nil {
		require.NoError(ts.T(), ts.API.db.DeleteUser(u), "Error deleting user")
	}

	u, err := models.NewUser("", email, "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")

	u.IsSuperAdmin = true
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	token, err := generateAccessToken(u, time.Second*time.Duration(ts.Config.JWT.Exp), ts.Config.JWT.Secret)
	require.NoError(ts.T(), err, "Error generating access token")

	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	require.NoError(ts.T(), err, "Error parsing token")

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers() {
	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=1>; rel=\"last\"", w.HeaderMap.Get("Link"))
	assert.Equal(ts.T(), "2", w.HeaderMap.Get("X-Total-Count"))

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	for _, user := range data["users"].([]interface{}) {
		assert.NotEmpty(ts.T(), user)
	}
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_Pagination() {
	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?per_page=1", nil)

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=2&per_page=1>; rel=\"next\", </admin/users?page=2&per_page=1>; rel=\"last\"", w.HeaderMap.Get("Link"))
	assert.Equal(ts.T(), "2", w.HeaderMap.Get("X-Total-Count"))

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	for _, user := range data["users"].([]interface{}) {
		assert.NotEmpty(ts.T(), user)
	}
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_SortAsc() {
	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	qv := req.URL.Query()
	qv.Set("sort", "created_at asc")
	req.URL.RawQuery = qv.Encode()

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 2)
	assert.Equal(ts.T(), "test1@example.com", data.Users[0].Email)
	assert.Equal(ts.T(), "test@example.com", data.Users[1].Email)
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_SortDesc() {
	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 2)
	assert.Equal(ts.T(), "test@example.com", data.Users[0].Email)
	assert.Equal(ts.T(), "test1@example.com", data.Users[1].Email)
}

// TestAdminUserCreate tests API /admin/user route (POST)
func (ts *AdminTestSuite) TestAdminUserCreate() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test1@example.com",
		"password": "test1",
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	u, err := ts.API.db.FindUserByEmailAndAudience("", "test1@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)

	data := models.User{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), u.Email, data.Email)
	assert.Equal(ts.T(), "email", data.AppMetaData["provider"])
}

// TestAdminUserGet tests API /admin/user route (GET)
func (ts *AdminTestSuite) TestAdminUserGet() {
	u, err := models.NewUser("", "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/user?email=test1@example.com&aud=%s", ts.Config.JWT.Aud), nil)

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["email"], "test1@example.com")
}

// TestAdminUserUpdate tests API /admin/user route (UPDATE)
func (ts *AdminTestSuite) TestAdminUserUpdate() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"role": "testing",
		"app_metadata": map[string]interface{}{
			"roles": []string{"writer", "editor"},
		},
		"user_metadata": map[string]interface{}{
			"name": "David",
		},
		"user": map[string]interface{}{
			"email": "test1@example.com",
			"aud":   ts.Config.JWT.Aud,
		},
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["role"], "testing")

	u, err := ts.API.db.FindUserByEmailAndAudience("", "test1@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.Equal(ts.T(), u.Role, "testing")
	assert.Equal(ts.T(), u.UserMetaData["name"], "David")
	assert.Len(ts.T(), u.AppMetaData["roles"], 2)
	assert.Contains(ts.T(), u.AppMetaData["roles"], "writer")
	assert.Contains(ts.T(), u.AppMetaData["roles"], "editor")
}

// TestAdminUserDelete tests API /admin/user route (DELETE)
func (ts *AdminTestSuite) TestAdminUserDelete() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"email": "test1@example.com",
			"aud":   ts.Config.JWT.Aud,
		},
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ts.makeSuperAdmin(req, "test@example.com")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)
}

// TestAdminUserCreateWithManagementToken tests API /admin/user route using the management token (POST)
func (ts *AdminTestSuite) TestAdminUserCreateWithManagementToken() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test2@example.com",
		"password": "test2",
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/user", &buffer)

	req.Header.Set("Authorization", "Bearer foobar")
	req.Header.Set("X-JWT-AUD", "op-test-aud")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	_, err := ts.API.db.FindUserByEmailAndAudience("", "test2@example.com", ts.Config.JWT.Aud)
	require.Error(ts.T(), err)

	u, err := ts.API.db.FindUserByEmailAndAudience("", "test2@example.com", "op-test-aud")
	require.NoError(ts.T(), err)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["email"], u.Email)
}

func TestAdmin(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
