package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

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

	user := &models.User{}
	if result := a.db.First(user, "email = ?", params.Email); result.Error != nil {
		if result.RecordNotFound() {
			NotFoundError(w, "User not found")
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	user.GenerateRecoveryToken()
	a.db.Save(user)

	if err := a.mailer.RecoveryMail(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error sending confirmation mail: %v", err))
		return
	}

	sendJSON(w, 200, &map[string]string{})
}
