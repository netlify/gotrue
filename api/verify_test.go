package api

import (
	//"net/http"
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestVerifySignup(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	// Find test user
	u, err := api.db.FindUserByEmailAndAudience("test@example.com", api.config.JWT.Aud)
	if err != nil {
		t.Error(err)
	}

	// Request body
	var buffer bytes.Buffer
	if err = json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"type":  "signup",
		"token": u.ConfirmationToken,
	}); err != nil {
		t.Error(err)
	}

	// Setup request
	req := httptest.NewRequest("POST", "http://localhost/verify", &buffer)
	req.Header.Set("Content-Type", "application/json")

	// Setup response recorder
	w := httptest.NewRecorder()
	ctx := req.Context()

	api.Verify(ctx, w, req)

	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Log(resp.Status)
		t.Fail()
	}
}
