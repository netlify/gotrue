package api

import (
	"context"
	"net/http"
)

// Index shows a description of the API
func (a *API) Index(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sendJSON(w, 200, map[string]string{
		"version":     a.version,
		"name":        "Netlify Auth",
		"description": "Netlify Auth is a user registration and authentication API",
	})
}
