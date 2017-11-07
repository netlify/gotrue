package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AdminTestSuite struct {
	suite.Suite
	User   *models.User
	API    *API
	Config *conf.Configuration

	token      string
	instanceID string
}

func TestAdmin(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &AdminTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}

	suite.Run(t, ts)
}

func (ts *AdminTestSuite) SetupTest() {
	test.CleanupTables()
}

func (ts *AdminTestSuite) makeSuperAdmin(email string) string {
	u, err := models.NewUser(ts.instanceID, email, "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": "Test User"})
	require.NoError(ts.T(), err, "Error making new user")

	u.IsSuperAdmin = true
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	data, err := ts.API.db.FindUserByInstanceIDAndID(ts.instanceID, u.ID)
	require.NoError(ts.T(), err, "Error checking admin user")

	token, err := generateAccessToken(data, time.Second*time.Duration(ts.Config.JWT.Exp), ts.Config.JWT.Secret)
	require.NoError(ts.T(), err, "Error generating access token")

	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	require.NoError(ts.T(), err, "Error parsing token")

	return token
}

func (ts *AdminTestSuite) makeSystemUser() string {
	u := models.NewSystemUser("", ts.Config.JWT.Aud)

	token, err := generateAccessToken(u, time.Second*time.Duration(ts.Config.JWT.Exp), ts.Config.JWT.Secret)
	require.NoError(ts.T(), err, "Error generating access token")

	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	require.NoError(ts.T(), err, "Error parsing token")

	return token
}

// TestAdminUsersUnauthorized tests API /admin/users route without authentication
func (ts *AdminTestSuite) TestAdminUsersUnauthorized() {
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	assert.Equal(ts.T(), w.Code, http.StatusUnauthorized)
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=1>; rel=\"last\"", w.HeaderMap.Get("Link"))
	assert.Equal(ts.T(), "1", w.HeaderMap.Get("X-Total-Count"))

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	for _, user := range data.Users {
		ts.NotNil(user)
		ts.Require().NotNil(user.UserMetaData)
		ts.Equal("Test User", user.UserMetaData["full_name"])
	}
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_Pagination() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	u, err = models.NewUser(ts.instanceID, "test2@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?per_page=1", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=2&per_page=1>; rel=\"next\", </admin/users?page=3&per_page=1>; rel=\"last\"", w.HeaderMap.Get("Link"))
	assert.Equal(ts.T(), "3", w.HeaderMap.Get("X-Total-Count"))

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	for _, user := range data["users"].([]interface{}) {
		assert.NotEmpty(ts.T(), user)
	}
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_SortAsc() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")

	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	// Hack the creation time to workaround GORM always setting the timestamps to now on create :(
	u.CreatedAt = time.Now().Add(5 * time.Minute)
	require.NoError(ts.T(), ts.API.db.UpdateUser(u), "Error updating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	qv := req.URL.Query()
	qv.Set("sort", "created_at asc")
	req.URL.RawQuery = qv.Encode()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

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

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_SortDesc() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	// Hack the creation time to workaround GORM always setting the timestamps to now on create :(
	u.CreatedAt = time.Now().Add(5 * time.Minute)
	require.NoError(ts.T(), ts.API.db.UpdateUser(u), "Error updating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

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

// TestAdminUserCreate tests API /admin/user route (POST)
func (ts *AdminTestSuite) TestAdminUserCreate() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test1@example.com",
		"password": "test1",
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users", &buffer)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := models.User{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	assert.Equal(ts.T(), "test1@example.com", data.Email)
	assert.Equal(ts.T(), "email", data.AppMetaData["provider"])
}

// TestAdminUserGet tests API /admin/user route (GET)
func (ts *AdminTestSuite) TestAdminUserGet() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/users/%s", u.ID), nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["email"], "test1@example.com")
	assert.NotNil(ts.T(), data["app_metadata"])
	assert.NotNil(ts.T(), data["user_metadata"])
}

// TestAdminUserUpdate tests API /admin/user route (UPDATE)
func (ts *AdminTestSuite) TestAdminUserUpdate() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"role": "testing",
		"app_metadata": map[string]interface{}{
			"roles": []string{"writer", "editor"},
		},
		"user_metadata": map[string]interface{}{
			"name": "David",
		},
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/users/%s", u.ID), &buffer)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := models.User{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), "testing", data.Role)
	assert.NotNil(ts.T(), data.UserMetaData)
	assert.Equal(ts.T(), "David", data.UserMetaData["name"])

	assert.NotNil(ts.T(), data.AppMetaData)
	assert.Len(ts.T(), data.AppMetaData["roles"], 2)
	assert.Contains(ts.T(), data.AppMetaData["roles"], "writer")
	assert.Contains(ts.T(), data.AppMetaData["roles"], "editor")
}

// TestAdminUserUpdate tests API /admin/user route (UPDATE) as system user
func (ts *AdminTestSuite) TestAdminUserUpdateAsSystemUser() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"role": "testing",
		"app_metadata": map[string]interface{}{
			"roles": []string{"writer", "editor"},
		},
		"user_metadata": map[string]interface{}{
			"name": "David",
		},
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/users/%s", u.ID), &buffer)

	token := ts.makeSystemUser()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["role"], "testing")

	u, err = ts.API.db.FindUserByEmailAndAudience(ts.instanceID, "test1@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.Equal(ts.T(), u.Role, "testing")
	assert.Equal(ts.T(), u.UserMetaData["name"], "David")
	assert.Len(ts.T(), u.AppMetaData["roles"], 2)
	assert.Contains(ts.T(), u.AppMetaData["roles"], "writer")
	assert.Contains(ts.T(), u.AppMetaData["roles"], "editor")
}

// TestAdminUserDelete tests API /admin/user route (DELETE)
func (ts *AdminTestSuite) TestAdminUserDelete() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	u, err := models.NewUser(ts.instanceID, "test-delete@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.CreateUser(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%s", u.ID), nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)
}

// TestAdminUserCreateWithManagementToken tests API /admin/user route using the management token (POST)
func (ts *AdminTestSuite) TestAdminUserCreateWithManagementToken() {
	ts.token = ts.makeSuperAdmin("test@example.com")
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test2@example.com",
		"password": "test2",
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users", &buffer)

	req.Header.Set("Authorization", "Bearer foobar")
	req.Header.Set("X-JWT-AUD", "op-test-aud")

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := models.User{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.NotNil(ts.T(), data.ID)
	assert.Equal(ts.T(), "test2@example.com", data.Email)
}
