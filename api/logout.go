package api

import (
	"context"
	"net/http"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	a.db.Logout(token.Claims["id"])

	w.WriteHeader(204)
}
