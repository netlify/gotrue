package api

import (
	"net/http"

	"github.com/netlify/gotrue/models"
	uuid "github.com/satori/go.uuid"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	claims := getClaims(ctx)

	a.clearCookieToken(ctx, w)

	if claims == nil {
		return badRequestError("Could not read User ID claim")
	}
	userID, err := uuid.FromString(claims.Subject)
	if err != nil {
		return badRequestError("Invalid User ID")
	}

	models.Logout(a.db, userID)
	w.WriteHeader(http.StatusNoContent)

	return nil
}
