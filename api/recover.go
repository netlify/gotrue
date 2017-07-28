package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/netlify/gotrue/models"
)

// RecoverParams holds the parameters for a password recovery request
type RecoverParams struct {
	Email string `json:"email"`
}

// Recover sends a recovery email
func (a *API) Recover(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
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
	user, err := a.db.FindUserByEmailAndAudience(params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	if user.RecoverySentAt.Add(a.config.Mailer.MaxFrequency).After(time.Now()) {
		user.GenerateRecoveryToken()
		if err := a.db.UpdateUser(user); err != nil {
			return internalServerError("Database error updating user").WithInternalError(err)
		}
		if err := a.mailer.RecoveryMail(user); err != nil {
			return internalServerError("Error sending recovery mail").WithInternalError(err)
		}
	}

	return sendJSON(w, http.StatusOK, &map[string]string{})
}
