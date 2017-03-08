package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/models"
)

// SignupParams are the parameters the Signup endpoint accepts
type SignupParams struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Data     map[string]interface{} `json:"data"`
}

// Signup is the endpoint for registering a new user
func (a *API) Signup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := &SignupParams{}

	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read Signup params: %v", err))
		return
	}

	if params.Email == "" || params.Password == "" {
		UnprocessableEntity(w, fmt.Sprintf("Signup requires a valid email and password"))
		return
	}

	user, err := a.db.FindUserByEmail(params.Email)
	if err != nil {
		if !models.IsNotFoundError(err) {
			InternalServerError(w, err.Error())
			return
		}

		user, err = a.signupNewUser(params)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}
	} else {
		user, err = a.confirmUser(user, params)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}
	}

	if err := a.mailer.ConfirmationMail(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error sending confirmation mail: %v", err))
		return
	}

	sendJSON(w, 200, user)
}

func (a *API) signupNewUser(params *SignupParams) (*models.User, error) {
	user, err := models.NewUser(params.Email, params.Password, params.Data)
	if err != nil {
		return nil, err
	}

	if err := a.db.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (a *API) confirmUser(user *models.User, params *SignupParams) (*models.User, error) {
	if user.IsRegistered() {
		return nil, errors.New("A user with this email address has already been registered")
	}

	user.UpdateUserMetaData(params.Data)
	user.GenerateConfirmationToken()

	if err := a.db.UpdateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}
