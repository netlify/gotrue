package api

import (
	"net/http"

	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	a.db.Where("user_id = ?", token.Claims["id"]).Delete(&models.RefreshToken{})

	w.WriteHeader(204)
}
