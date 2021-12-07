package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
)

// SignupParams are the parameters the Signup endpoint accepts
type SignupParams struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Data     map[string]interface{} `json:"data"`
	Provider string                 `json:"-"`
	Aud      string                 `json:"-"`
}

// Signup is the endpoint for registering a new user
func (a *API) Signup(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)

	if config.DisableSignup {
		return forbiddenError("Signups not allowed for this instance")
	}

	params := &SignupParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read Signup params: %v", err)
	}

	if params.Password == "" {
		return unprocessableEntityError("Signup requires a valid password")
	}
	if err := a.validateEmail(ctx, params.Email); err != nil {
		return err
	}

	instanceID := getInstanceID(ctx)
	params.Aud = a.requestAud(ctx, r)
	user, err := models.FindUserByEmailAndAudience(a.db, instanceID, params.Email, params.Aud)
	if err != nil && !models.IsNotFoundError(err) {
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	err = a.db.Transaction(func(tx *storage.Connection) error {
		var terr error
		if user != nil {
			if user.IsConfirmed() {
				return badRequestError("A user with this email address has already been registered")
			}

			if err := user.UpdateUserMetaData(tx, params.Data); err != nil {
				return internalServerError("Database error updating user").WithInternalError(err)
			}
		} else {
			params.Provider = "email"
			user, terr = a.signupNewUser(ctx, tx, params)
			if terr != nil {
				return terr
			}
		}

		if config.Mailer.Autoconfirm {
			if terr = models.NewAuditLogEntry(tx, instanceID, user, models.UserSignedUpAction, nil); terr != nil {
				return terr
			}
			if terr = triggerEventHooks(ctx, tx, SignupEvent, user, instanceID, config); terr != nil {
				return terr
			}
			if terr = user.Confirm(tx); terr != nil {
				return internalServerError("Database error updating user").WithInternalError(terr)
			}
		} else {
			mailer := a.Mailer(ctx)
			referrer := a.getReferrer(r)
			if terr = sendConfirmation(tx, user, mailer, config.SMTP.MaxFrequency, referrer); terr != nil {
				return internalServerError("Error sending confirmation mail").WithInternalError(terr)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, user)
}

func (a *API) signupNewUser(ctx context.Context, conn *storage.Connection, params *SignupParams) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)

	user, err := models.NewUser(instanceID, params.Email, params.Password, params.Aud, params.Data)
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

	err = conn.Transaction(func(tx *storage.Connection) error {
		if terr := tx.Create(user); terr != nil {
			return internalServerError("Database error saving new user").WithInternalError(terr)
		}
		if terr := user.SetRole(tx, config.JWT.DefaultGroupName); terr != nil {
			return internalServerError("Database error updating user").WithInternalError(terr)
		}
		if terr := triggerEventHooks(ctx, tx, ValidateEvent, user, instanceID, config); terr != nil {
			return terr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}
