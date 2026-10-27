package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	jwt "github.com/golang-jwt/jwt/v4"
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

	token      string
	instanceID uuid.UUID
}

func TestAdmin(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &AdminTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *AdminTestSuite) SetupTest() {
	require.NoError(ts.T(), models.TruncateAll(ts.API.db))
	ts.Config.External.Email.Disabled = false
	ts.token = ts.makeSuperAdmin("test@example.com")
}

func (ts *AdminTestSuite) makeSuperAdmin(email string) string {
	u, err := models.NewUser(ts.instanceID, email, "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": "Test User"})
	require.NoError(ts.T(), err, "Error making new user")

	u.IsSuperAdmin = true
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	token, err := generateAccessToken(u, time.Second*time.Duration(ts.Config.JWT.Exp), ts.Config.JWT.Secret)
	require.NoError(ts.T(), err, "Error generating access token")

	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.Config.JWT.Secret), nil
	})
	require.NoError(ts.T(), err, "Error parsing token")

	return token
}

func (ts *AdminTestSuite) makeSystemUser() string {
	u := models.NewSystemUser(uuid.Nil, ts.Config.JWT.Aud)

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
	assert.Equal(ts.T(), http.StatusUnauthorized, w.Code)
}

func (ts *AdminTestSuite) TestAdminUserLookup() {
	user, err := models.NewUser(ts.instanceID, "lookup@example.com", "test-password", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err)
	user.ConfirmationToken = "test-confirmation-token"
	user.RecoveryToken = "test-recovery-token"
	require.NoError(ts.T(), ts.API.db.Create(user))

	for _, scope := range []struct {
		instanceID uuid.UUID
		audience   string
	}{
		{uuid.Must(uuid.NewV4()), ts.Config.JWT.Aud},
		{ts.instanceID, "other-audience"},
	} {
		other, err := models.NewUser(scope.instanceID, "outside@example.com", "test-password", scope.audience, nil)
		require.NoError(ts.T(), err)
		require.NoError(ts.T(), ts.API.db.Create(other))
	}
	userToken, err := generateAccessToken(user, time.Hour, ts.Config.JWT.Secret)
	require.NoError(ts.T(), err)

	for _, tc := range []struct {
		name   string
		body   string
		query  string
		token  string
		status int
	}{
		{"exact email", `{"email":"lookup@example.com"}`, "", ts.token, http.StatusOK},
		{"missing user", `{"email":"missing@example.com"}`, "", ts.token, http.StatusNotFound},
		{"different instance or audience", `{"email":"outside@example.com"}`, "", ts.token, http.StatusNotFound},
		{"partial email", `{"email":"lookup"}`, "", ts.token, http.StatusNotFound},
		{"substring email", `{"email":"ookup@example.com"}`, "", ts.token, http.StatusNotFound},
		{"wildcard is not a pattern", `{"email":"%@example.com"}`, "", ts.token, http.StatusNotFound},
		{"missing email", `{}`, "", ts.token, http.StatusBadRequest},
		{"empty email", `{"email":""}`, "", ts.token, http.StatusBadRequest},
		{"display name", `{"email":"Test User <lookup@example.com>"}`, "", ts.token, http.StatusNotFound},
		{"filter", `{"email":"lookup@example.com","filter":"lookup"}`, "", ts.token, http.StatusBadRequest},
		{"pagination", `{"email":"lookup@example.com","page":1}`, "", ts.token, http.StatusBadRequest},
		{"query parameters", `{"email":"lookup@example.com"}`, "?page=1", ts.token, http.StatusBadRequest},
		{"body audience", `{"email":"lookup@example.com","aud":"other-audience"}`, "", ts.token, http.StatusBadRequest},
		{"multiple objects", `{"email":"lookup@example.com"} {}`, "", ts.token, http.StatusBadRequest},
		{"unauthenticated", `{"email":"lookup@example.com"}`, "", "", http.StatusUnauthorized},
		{"non-admin", `{"email":"lookup@example.com"}`, "", userToken, http.StatusUnauthorized},
	} {
		ts.Run(tc.name, func() {
			req := httptest.NewRequest(http.MethodPost, "/admin/users/lookup"+tc.query, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), tc.status, w.Code, w.Body.String())
			assert.Empty(ts.T(), w.Header().Get("Link"))
			assert.Empty(ts.T(), w.Header().Get("X-Total-Count"))
			if tc.status == http.StatusOK {
				var body map[string]interface{}
				require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&body))
				assert.Equal(ts.T(), user.ID.String(), body["id"])
				assert.Equal(ts.T(), user.Email, body["email"])
				for _, field := range []string{"users", "encrypted_password", "confirmation_token", "recovery_token", "email_change_token"} {
					assert.NotContains(ts.T(), body, field)
				}
			}
		})
	}
}

func TestAdminUserLookupMalformedBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/users/lookup", strings.NewReader(`{`))
	err := (&API{}).adminUserLookup(httptest.NewRecorder(), req)
	require.EqualError(t, err, "400: Expected a JSON object containing only email")
}

func (ts *AdminTestSuite) TestAdminUserLookupAudienceHeader() {
	for _, audience := range []string{ts.Config.JWT.Aud, "other-audience"} {
		user, err := models.NewUser(ts.instanceID, "shared@example.com", "test-password", audience, nil)
		require.NoError(ts.T(), err)
		require.NoError(ts.T(), ts.API.db.Create(user))
	}
	for _, audience := range []string{ts.Config.JWT.Aud, "other-audience", "missing-audience"} {
		ts.Run(audience, func() {
			req := httptest.NewRequest(http.MethodPost, "/admin/users/lookup", strings.NewReader(`{"email":"shared@example.com"}`))
			req.Header.Set("Authorization", "Bearer "+ts.token)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-JWT-AUD", audience)
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			if audience == "missing-audience" {
				require.Equal(ts.T(), http.StatusNotFound, w.Code, w.Body.String())
				return
			}
			require.Equal(ts.T(), http.StatusOK, w.Code, w.Body.String())
			var user models.User
			require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&user))
			assert.Equal(ts.T(), audience, user.Aud)
			assert.Equal(ts.T(), "shared@example.com", user.Email)
		})
	}
}

func (ts *AdminTestSuite) TestAdminUserLookupStoredEmailFormats() {
	for _, email := range []string{
		" leading@example.com",
		`"test user"@example.com`,
		"Test User <display@example.com>",
		"provider-email-without-at-sign",
	} {
		ts.Run(email, func() {
			user, err := models.NewUser(ts.instanceID, email, "test-password", ts.Config.JWT.Aud, nil)
			require.NoError(ts.T(), err)
			require.NoError(ts.T(), ts.API.db.Create(user))
			body, err := json.Marshal(map[string]string{"email": email})
			require.NoError(ts.T(), err)
			req := httptest.NewRequest(http.MethodPost, "/admin/users/lookup", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+ts.token)
			w := httptest.NewRecorder()
			ts.API.handler.ServeHTTP(w, req)
			require.Equal(ts.T(), http.StatusOK, w.Code, w.Body.String())
			var result models.User
			require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&result))
			assert.Equal(ts.T(), user.ID, result.ID)
			assert.Equal(ts.T(), email, result.Email)
		})
	}
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers() {
	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=1>; rel=\"last\"", w.Result().Header.Get("Link"))
	assert.Equal(ts.T(), "1", w.Result().Header.Get("X-Total-Count"))

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
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	u, err = models.NewUser(ts.instanceID, "test2@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?per_page=1", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	assert.Equal(ts.T(), "</admin/users?page=2&per_page=1>; rel=\"next\", </admin/users?page=3&per_page=1>; rel=\"last\"", w.Result().Header.Get("Link"))
	assert.Equal(ts.T(), "3", w.Result().Header.Get("X-Total-Count"))

	data := make(map[string]interface{})
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))
	for _, user := range data["users"].([]interface{}) {
		assert.NotEmpty(ts.T(), user)
	}
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_SortAsc() {
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")

	// if the created_at times are the same, then the sort order is not guaranteed
	time.Sleep(1 * time.Second)
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

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
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	// if the created_at times are the same, then the sort order is not guaranteed
	time.Sleep(1 * time.Second)
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

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

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_FilterEmail() {
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?filter=test1", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 1)
	assert.Equal(ts.T(), "test1@example.com", data.Users[0].Email)
}

// TestAdminUsers tests API /admin/users route
func (ts *AdminTestSuite) TestAdminUsers_FilterName() {
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?filter=User", nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)

	data := struct {
		Users []*models.User `json:"users"`
		Aud   string         `json:"aud"`
	}{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	require.Len(ts.T(), data.Users, 1)
	assert.Equal(ts.T(), "test@example.com", data.Users[0].Email)
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
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": "Test Get User"})
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

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
	md := data["user_metadata"].(map[string]interface{})
	assert.Len(ts.T(), md, 1)
	assert.Equal(ts.T(), "Test Get User", md["full_name"])
}

// TestAdminUserUpdate tests API /admin/user route (UPDATE)
func (ts *AdminTestSuite) TestAdminUserUpdate() {
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

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
	u, err := models.NewUser(ts.instanceID, "test1@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

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

	u, err = models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, "test1@example.com", ts.Config.JWT.Aud)
	require.NoError(ts.T(), err)
	assert.Equal(ts.T(), u.Role, "testing")
	require.NotNil(ts.T(), u.UserMetaData)
	require.Contains(ts.T(), u.UserMetaData, "name")
	assert.Equal(ts.T(), u.UserMetaData["name"], "David")
	require.NotNil(ts.T(), u.AppMetaData)
	require.Contains(ts.T(), u.AppMetaData, "roles")
	assert.Len(ts.T(), u.AppMetaData["roles"], 2)
	assert.Contains(ts.T(), u.AppMetaData["roles"], "writer")
	assert.Contains(ts.T(), u.AppMetaData["roles"], "editor")
}

// TestAdminUserDelete tests API /admin/user route (DELETE)
func (ts *AdminTestSuite) TestAdminUserDelete() {
	u, err := models.NewUser(ts.instanceID, "test-delete@example.com", "test", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error making new user")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error creating user")

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/users/%s", u.ID), nil)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

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

func (ts *AdminTestSuite) TestAdminUserCreateWithDisabledEmailLogin() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test1@example.com",
		"password": "test1",
	}))

	// Setup request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/users", &buffer)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ts.token))

	ts.Config.External.Email.Disabled = true

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusBadRequest, w.Code)
}
