package api

import (
	"net/http"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	claims := getClaims(ctx)

	a.clearCookieToken(ctx, w)

	if claims == nil || claims.Subject == "" {
		return badRequestError("Could not read User ID claim")
	}

	a.db.Logout(claims.Subject)
	w.WriteHeader(http.StatusNoContent)

	return nil
}
