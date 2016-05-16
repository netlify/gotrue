package api

import (
	"net/http"

	"golang.org/x/net/context"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {

}
