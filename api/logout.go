package api

import (
	"net/http"

	"github.com/netlify/gotrue/models"
	"context"
)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(ctx)

	a.clearCookieToken(ctx, w)

	u, err := getUserFromClaims(ctx, a.db)
	if err != nil {
		return unauthorizedError("Invalid user").WithInternalError(err)
	}

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		if terr := models.NewAuditLogEntry(ctx, a.db, instanceID, u, models.LogoutAction, nil); terr != nil {
			return terr
		}
		return models.Logout(ctx, a.db, instanceID, u.ID)
	})
	if err != nil {
		return internalServerError("Error logging out user").WithInternalError(err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
