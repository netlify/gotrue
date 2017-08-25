package api

import (
	"net/http"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil || claims.ID == "" {
		return badRequestError("Could not read User ID claim")
	}

	a.db.Logout(claims.ID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
