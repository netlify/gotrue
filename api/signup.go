package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/netlify/gotrue/models"
)

// SignupParams are the parameters the Signup endpoint accepts
type SignupParams struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Data     map[string]interface{} `json:"data"`
	Provider string
}

// Signup is the endpoint for registering a new user
func (a *API) Signup(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)

	if config.DisableSignup {
		return forbiddenError("Signups not allowed for this instance")
	}

	instanceID := getInstanceID(ctx)
	params := &SignupParams{}

	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read Signup params: %v", err)
	}

	if params.Email == "" || params.Password == "" {
		return unprocessableEntityError("Signup requires a valid email and password")
	}

	mailer := getMailer(ctx)
	aud := a.requestAud(ctx, r)

	user, err := a.db.FindUserByEmailAndAudience(instanceID, params.Email, aud)
	if err != nil {
		if !models.IsNotFoundError(err) {
			return internalServerError("Database error finding user").WithInternalError(err)
		}
		if err = mailer.ValidateEmail(params.Email); err != nil {
			return unprocessableEntityError("Unable to validate email address: " + err.Error())
		}

		params.Provider = "email"
		user, err = a.signupNewUser(ctx, params, aud)
		if err != nil {
			return err
		}
	} else {
		err = a.updateUserMetadata(user, params)
		if err != nil {
			return err
		}
	}

	if config.Mailer.Autoconfirm {
		if config.Webhook.HasEvent("signup") {
			if err := triggerHook(SignupEvent, user, instanceID, config); err != nil {
				return err
			}
			a.db.UpdateUser(user)
		}
		user.Confirm()
	} else {
		if err := mailer.ConfirmationMail(user); err != nil {
			return internalServerError("Error sending confirmation mail").WithInternalError(err)
		}
		now := time.Now()
		user.ConfirmationSentAt = &now
	}

	if err = a.db.UpdateUser(user); err != nil {
		return internalServerError("Database error updating user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

func (a *API) signupNewUser(ctx context.Context, params *SignupParams, aud string) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)

	user, err := models.NewUser(instanceID, params.Email, params.Password, aud, params.Data)
	if err != nil {
		return nil, internalServerError("Database error creating user").WithInternalError(err)
	}
	if user.AppMetaData == nil {
		user.AppMetaData = make(map[string]interface{})
	}
	user.AppMetaData["provider"] = params.Provider

	if params.Password == "" {
		user.EncryptedPassword = ""
	}

	user.SetRole(config.JWT.DefaultGroupName)

	if config.Webhook.HasEvent("validate") {
		if err := triggerHook(ValidateEvent, user, instanceID, config); err != nil {
			return nil, err
		}
	}

	if err := a.db.CreateUser(user); err != nil {
		return nil, internalServerError("Database error saving new user").WithInternalError(err)
	}

	return user, nil
}

func (a *API) updateUserMetadata(user *models.User, params *SignupParams) error {
	if user.IsConfirmed() {
		return badRequestError("A user with this email address has already been registered")
	}

	user.UpdateUserMetaData(params.Data)
	if err := a.db.UpdateUser(user); err != nil {
		return internalServerError("Database error updating user").WithInternalError(err)
	}
	return nil
}
