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
)

// TestAdminUsersUnauthorized tests API /admin/users route without authentication
func TestAdminUsersUnauthorized(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	// Setup request
	req := httptest.NewRequest("GET", "/admin/users", nil)

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	api.adminUsers(ctx, w, req)

	resp := w.Result()

	if resp.StatusCode == 200 {
		t.Log(resp)
		t.Fail()
	}
}

func makeSuperAdmin(req *http.Request, api *API, email string, t *testing.T) (context.Context, *httptest.ResponseRecorder) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	// Cleanup existing user
	u, err := api.db.FindUserByEmailAndAudience(email, api.config.JWT.Aud)
	if err == nil {
		if err = api.db.DeleteUser(u); err != nil {
			t.Error(err)
		}
	}

	u, err = models.NewUser(email, "test", api.config.JWT.Aud, nil)
	if err != nil {
		t.Error(err)
	}

	u.IsSuperAdmin = true
	api.db.CreateUser(u)

	token, err := api.generateAccessToken(u)
	if err != nil {
		t.Error(err)
	}

	tok, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if token.Header["alg"] != "HS256" {
			t.Error("Invalid alg")
		}

		return []byte(api.config.JWT.Secret), nil
	})
	if err != nil {
		t.Error(err)
	}

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	return context.WithValue(ctx, "jwt", tok), w
}

// TestAdminUsers tests API /admin/users route
func TestAdminUsers(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	// Setup request
	req := httptest.NewRequest("GET", "/admin/users", nil)

	// Setup response recorder with super admin privileges
	ctx, w := makeSuperAdmin(req, api, "test@example.com", t)

	api.adminUsers(ctx, w, req)

	resp := w.Result()

	data := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}

	if resp.StatusCode != 200 {
		t.Log(resp)
		t.Fail()
	}

	if len(data["users"].([]interface{})) < 1 {
		t.Error("Invalid user list")
	}
}

// TestAdminUserCreate tests API /admin/user route (POST)
func TestAdminUserCreate(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	var buffer bytes.Buffer
	err = json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test1@example.com",
		"password": "test1",
	})
	if err != nil {
		t.Error(err)
	}

	// Setup request
	req := httptest.NewRequest("POST", "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ctx, w := makeSuperAdmin(req, api, "test@example.com", t)

	api.adminUserCreate(ctx, w, req)

	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Error(resp)
		t.Fail()
		return
	}

	u, err := api.db.FindUserByEmailAndAudience("test1@example.com", api.config.JWT.Aud)
	if err != nil {
		t.Error(err)
		return
	}

	data := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}

	if data["email"] != u.Email {
		t.Error("Invalid email address")
	}
}

// TestAdminUserGet tests API /admin/user route (GET)
func TestAdminUserGet(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	var buffer bytes.Buffer
	json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"email": "test1@example.com",
			"aud":   api.config.JWT.Aud,
		},
	})

	// Setup request
	req := httptest.NewRequest("GET", "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ctx, w := makeSuperAdmin(req, api, "test@example.com", t)

	api.adminUserGet(ctx, w, req)

	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Log(resp)
		t.Fail()
	}

	data := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}

	if data["email"] != "test1@example.com" {
		t.Error("Invalid email address: ", data)
	}

}

// TestAdminUserDelete tests API /admin/user route (DELETE)
func TestAdminUserDelete(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	var buffer bytes.Buffer
	json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"email": "test1@example.com",
			"aud":   api.config.JWT.Aud,
		},
	})

	// Setup request
	req := httptest.NewRequest("DELETE", "/admin/user", &buffer)

	// Setup response recorder with super admin privileges
	ctx, w := makeSuperAdmin(req, api, "test@example.com", t)

	api.adminUserDelete(ctx, w, req)

	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Log(resp)
		t.Fail()
	}
}
