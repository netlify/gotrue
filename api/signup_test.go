package api

import (
	//"net/http"
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestSignup tests API /signup route
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

// TestSignupTwice checks to make sure the same email cannot be registered twice
func TestSignupTwice(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	// Request body
	var buffer bytes.Buffer

	encode := func() {
		if err = json.NewEncoder(&buffer).Encode(map[string]interface{}{
			"email":    "test1@example.com",
			"password": "test1",
			"data": map[string]interface{}{
				"a": 1,
			},
		}); err != nil {
			t.Error(err)
		}
	}

	encode()

	// Setup request
	req := httptest.NewRequest("POST", "http://localhost/signup", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	y := httptest.NewRecorder()
	ctx := req.Context()

	api.Signup(ctx, y, req)
	u, err := api.db.FindUserByEmailAndAudience("test1@example.com", api.config.JWT.Aud)
	if err == nil {
		u.Confirm()
		api.db.UpdateUser(u)
	}

	encode()
	api.Signup(ctx, w, req)

	resp := w.Result()

	data := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}

	if code, ok := data["code"]; ok {
		if c, ok := code.(float64); ok {
			if resp.StatusCode != 500 || c != 500 {
				t.Log("StatusCode: ", resp.StatusCode)
				t.Log("Code: ", c)
				t.Log("Message: ", data["msg"])
				t.Fail()
			}
		} else {
			t.Error("Invalid value type for 'code'")
		}
	} else {
		t.Error("Invalid value for 'code'")
	}
}
