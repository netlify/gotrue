package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/netlify/netlify-auth/models"
)

// UserUpdateParams parameters for updating a user
type UserUpdateParams struct {
	Email            string                 `json:"email"`
	Password         string                 `json:"password"`
	EmailChangeToken string                 `json:"email_change_token"`
	Data             map[string]interface{} `json:"data"`
	AppData          map[string]interface{} `json:"app_metadata,omitempty"`
}

// UserGet returns a user
func (a *API) UserGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	id, ok := token.Claims["id"].(string)
	if !ok {
		BadRequestError(w, "Could not read User ID claim")
		return
	}

	user, err := a.db.FindUserByID(id)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	sendJSON(w, 200, user)
}

// UserUpdate updates fields on a user
func (a *API) UserUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	params := &UserUpdateParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read User Update params: %v", err))
		return
	}

	id, ok := token.Claims["id"].(string)
	if !ok {
		BadRequestError(w, "Could not read User ID claim")
		return
	}

	user, err := a.db.FindUserByID(id)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	var sendChangeEmailVerification bool
	if params.Email != "" {
		exists, err := a.db.IsDuplicatedEmail(params.Email, user.ID)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}

		if exists {
			UnprocessableEntity(w, "Email address already registered by another user")
			return
		}

		user.GenerateEmailChange(params.Email)
		sendChangeEmailVerification = true
	}

	logrus.Debugf("Checkign params for token %v", params)

	if params.EmailChangeToken != "" {
		logrus.Debugf("Got change token %v", params.EmailChangeToken)

		if params.EmailChangeToken != user.EmailChangeToken {
			UnauthorizedError(w, "Email Change Token didn't match token on file")
			return
		}

		user.ConfirmEmailChange()
	}

	if params.Password != "" {
		if err = user.EncryptPassword(params.Password); err != nil {
			InternalServerError(w, fmt.Sprintf("Error during password encryption: %v", err))
			return
		}
	}

	if params.Data != nil {
		user.UpdateUserMetaData(params.Data)
	}

	if params.AppData != nil {
		if !user.HasRole(a.config.JWT.AdminGroupName) {
			UnauthorizedError(w, "Updating app_metadata requires admin privileges")
			return
		}

		user.UpdateAppMetaData(params.AppData)
	}

	if err := a.db.UpdateUser(user); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	if sendChangeEmailVerification {
		a.mailer.EmailChangeMail(user)
	}

	sendJSON(w, 200, user)
}
