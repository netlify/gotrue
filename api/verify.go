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
	Type  string `json:"type"`
	Token string `json:"token"`
}

func queryForParams(params *VerifyParams) string {
	switch params.Type {
	case "signup":
		return "confirmation_token = ?"
	case "recovery":
		return "recovery_token = ?"
	}
	return ""
}

// Verify exchanges a confirmation or recovery token to a refresh token
func (a *API) Verify(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := &VerifyParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read verification params: %v", err))
		return
	}

	query := queryForParams(params)

	if query == "" || params.Token == "" {
		UnprocessableEntity(w, fmt.Sprintf("Verify requires a token and a type"))
		return
	}

	user := &models.User{}

	if result := a.db.First(user, query, params.Token); result.Error != nil {
		if result.RecordNotFound() {
			NotFoundError(w, "Invalid token")
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	switch params.Type {
	case "signup":
		user.Confirm()
	case "recover":
		user.Recover()
	}

	tx := a.db.Begin()
	if err := a.db.Save(user).Error; err != nil {
		tx.Rollback()
		InternalServerError(w, fmt.Sprintf("Error confirming user: %v", err))
		return
	}

	a.issueRefreshToken(tx, user, w)
}
