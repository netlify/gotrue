package api

import (
	"net/http"
	//"bytes"
	"context"
	"encoding/json"
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
	req := httptest.NewRequest("GET", "http://localhost/admin/users", nil)

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
	u.Role = "admin"
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
	req := httptest.NewRequest("GET", "http://localhost/admin/users", nil)

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
