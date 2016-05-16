package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/authlify/models"
	"golang.org/x/net/context"
)

// SignupParams are the parameters the Signup endpoint accepts
type SignupParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func validParams(params *SignupParams) {
	return params.Email != "" && params.Password != ""
}

// Signup is the endpoint for registering a new user
func (a *API) Signup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var user *models.User
	params := &SignupParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read Signup params: %v", err))
		return
	}

	if !validParams(params) {
		UnprocessableEntity(w, fmt.Sprintf("Signup requires a valid email and password"))
		return
	}

	existingUser := &models.User{}
	a.db.First(existingUser, "email = ?", params.Email)
	if existingUser.ID != 0 {
		if existingUser.ConfirmationSentAt.IsZero() {
			existingUser.GenerateConfirmationToken()
			user = existingUser
		} else {
			UnprocessableEntity(w, fmt.Sprintf("A user with this email address has already been registered"))
			return
		}
	} else {
		user, err = models.CreateUser(params.Email, params.Password)
		if err != nil {
			InternalServerError(w, fmt.Sprintf("Error creating user: %v", err))
			return
		}
	}

	if err := a.mailer.ConfirmationMail(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error sending confirmation mail: %v", err))
		return
	}

	sendJSON(w, 200, user)
}
