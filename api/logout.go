package api

import (
	"net/http"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	token := getToken(ctx)

	id, ok := token.Claims["id"].(string)
	if !ok {
		return badRequestError("Could not read User ID claim")
	}

	a.db.Logout(id)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
