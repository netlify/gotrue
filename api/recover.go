package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/models"
)

// RecoverParams holds the parameters for a password recovery request
type RecoverParams struct {
	Email string `json:"email"`
}

// Recover sends a recovery email
func (a *API) Recover(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := &RecoverParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read verification params: %v", err))
		return
	}

	if params.Email == "" {
		UnprocessableEntity(w, fmt.Sprintf("Password recovery requires an email"))
		return
	}

	aud := a.requestAud(r)
	user, err := a.db.FindUserByEmailAndAudience(params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	user.GenerateRecoveryToken()
	if err := a.db.UpdateUser(user); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	if err := a.mailer.RecoveryMail(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error sending confirmation mail: %v", err))
		return
	}

	sendJSON(w, 200, &map[string]string{})
}
