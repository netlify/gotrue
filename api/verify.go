package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

// VerifyParams are the parameters the Verify endpoint accepts
type VerifyParams struct {
	ConfirmationToken string `json:"token"`
}

func (a *API) Verify(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := &VerifyParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read verification params: %v", err))
		return
	}

	if params.ConfirmationToken == "" {
		UnprocessableEntity(w, fmt.Sprintf("Verify requires a confirmation token"))
		return
	}

	user := &models.User{}
	if result := a.db.First(user, "confirmation_token = ?", params.ConfirmationToken); result.Error != nil {
		if result.RecordNotFound() {
			NotFoundError(w, "Confirmation token not found")
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	user.Confirm()
	a.db.Save(user)

	sendJSON(w, 200, user)
}
