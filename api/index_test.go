package api

import (
	//"net/http"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestIndex tests API / route
func TestIndex(t *testing.T) {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	if err != nil {
		t.Error(err)
	}
	defer api.db.Close()

	// Setup request and response reader
	req := httptest.NewRequest("GET", "http://localhost/", nil)
	w := httptest.NewRecorder()
	ctx := req.Context()
	api.Index(ctx, w, req)

	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Fail()
	}

	// Check response data
	data := make(map[string]string)
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}

	if data["name"] != "GoTrue" || data["version"] != "v1" {
		t.Error("Invalid response data")
	}

}
