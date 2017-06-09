package api

import (
	//"net/http"
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestIndex tests API / route
func TestSignup(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	// Cleanup existing user
	u, err := api.db.FindUserByEmailAndAudience("test@example.com", api.config.JWT.Aud)
	if err == nil {
		if err = api.db.DeleteUser(u); err != nil {
			t.Error(err)
		}
	}

	// Request body
	var buffer bytes.Buffer
	if err = json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test@example.com",
		"password": "test",
		"data": map[string]interface{}{
			"a": 1,
		},
	}); err != nil {
		t.Error(err)
	}

	// Setup request
	req := httptest.NewRequest("POST", "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	api.Signup(ctx, w, req)

	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Log(resp)
		t.Fail()
	}

	data := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}
}
