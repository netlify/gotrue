package api

import (
	"net/http"
	"regexp"

	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

var bearerRegexp = regexp.MustCompile(`^(?:B|b)earer (\S+$)`)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	a.db.Where("user_id = ?", token.Claims["id"]).Delete(&models.RefreshToken{})

	w.WriteHeader(204)
}
