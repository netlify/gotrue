package api

import (
	"encoding/json"
	"net/http"

	"github.com/netlify/gotrue/models"
	"context"
)

// RecoverParams holds the parameters for a password recovery request
type RecoverParams struct {
	Email string `json:"email"`
}

// Recover sends a recovery email
func (a *API) Recover(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)
	instanceID := getInstanceID(ctx)
	params := &RecoverParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read verification params: %v", err)
	}

	if params.Email == "" {
		return unprocessableEntityError("Password recovery requires an email")
	}

	aud := a.requestAud(ctx, r)
	user, err := models.FindUserByEmailAndAudience(r.Context(), a.db, instanceID, params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		if terr := models.NewAuditLogEntry(ctx, a.db, instanceID, user, models.UserRecoveryRequestedAction, nil); terr != nil {
			return terr
		}

		mailer := a.Mailer(ctx)
		referrer := a.getReferrer(r)
		return a.sendPasswordRecovery(ctx, a.db, user, mailer, config.SMTP.MaxFrequency, referrer)
	})
	if err != nil {
		return internalServerError("Error recovering user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, &map[string]string{})
}
