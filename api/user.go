package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/log"
	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
)

const (
	emailChangeType = "email_change"
)

// UserUpdateParams parameters for updating a user
type UserUpdateParams struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Data     map[string]interface{} `json:"data"`
	AppData  map[string]interface{} `json:"app_metadata,omitempty"`
}
type EmailChangeParams struct {
	EmailChangeToken string `json:"email_change_token"`
}

// UserGet responsible for getting user info or confirm updated fields
func (a *API) UserGet(w http.ResponseWriter, r *http.Request) error {
	userChangeType := r.FormValue("type")
	switch userChangeType {
	case emailChangeType:
		return a.userEmailChangeVerify(w, r)
	default:
		return a.userInfo(w, r)
	}
}

func (a *API) userInfo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		return badRequestError("Could not read claims")
	}

	userID, err := uuid.FromString(claims.Subject)
	if err != nil {
		return badRequestError("Could not read User ID claim")
	}

	aud := a.requestAud(ctx, r)
	if aud != claims.Audience {
		return badRequestError("Token audience doesn't match request audience")
	}

	user, err := models.FindUserByID(a.db, userID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

func (a *API) userEmailChangeVerify(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	params := &EmailChangeParams{}
	params.EmailChangeToken = r.FormValue("email_change_token")
	if params.EmailChangeToken == "" {
		return badRequestError("Could not found email_change_token")
	}

	claims := getClaims(ctx)
	userID, err := uuid.FromString(claims.Subject)
	if err != nil {
		return badRequestError("Could not read User ID claim")
	}

	log.Debugf("Got change token %v", params.EmailChangeToken)

	user, err := models.FindUserByID(a.db, userID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	err = a.db.Transaction(func(tx *storage.Connection) error {
		var terr error

		if params.EmailChangeToken != user.EmailChangeToken {
			return unauthorizedError("Email Change Token didn't match token on file")
		}

		if terr = user.ConfirmEmailChange(tx); terr != nil {
			return internalServerError("Error updating user").WithInternalError(terr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, user)
}

// UserUpdate updates fields on a user
func (a *API) UserUpdate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)
	instanceID := getInstanceID(ctx)

	params := &UserUpdateParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read User Update params: %v", err)
	}

	claims := getClaims(ctx)
	userID, err := uuid.FromString(claims.Subject)
	if err != nil {
		return badRequestError("Could not read User ID claim")
	}

	user, err := models.FindUserByID(a.db, userID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	log := getLogEntry(r)
	log.Debugf("Checking params for token %v", params)

	err = a.db.Transaction(func(tx *storage.Connection) error {
		var terr error
		if params.Password != "" {
			if len(params.Password) < config.PasswordMinLength {
				return unprocessableEntityError(fmt.Sprintf("Password should be at least %d characters", config.PasswordMinLength))
			}

			if terr = user.UpdatePassword(tx, params.Password); terr != nil {
				return internalServerError("Error during password storage").WithInternalError(terr)
			}
		}

		if params.Data != nil {
			if terr = user.UpdateUserMetaData(tx, params.Data); terr != nil {
				return internalServerError("Error updating user").WithInternalError(terr)
			}
		}

		if params.AppData != nil {
			if !a.isAdmin(ctx, user, config.JWT.Aud) {
				return unauthorizedError("Updating app_metadata requires admin privileges")
			}

			if terr = user.UpdateAppMetaData(tx, params.AppData); terr != nil {
				return internalServerError("Error updating user").WithInternalError(terr)
			}
		}

		if params.Email != "" && params.Email != user.Email {
			if terr = a.validateEmail(ctx, params.Email); terr != nil {
				return terr
			}

			var exists bool
			if exists, terr = models.IsDuplicatedEmail(tx, instanceID, params.Email, user.Aud); terr != nil {
				return internalServerError("Database error checking email").WithInternalError(terr)
			} else if exists {
				return unprocessableEntityError("Email address already registered by another user")
			}

			mailer := a.Mailer(ctx)
			referrer := a.getReferrer(r)
			if terr = a.sendEmailChange(tx, user, mailer, params.Email, referrer); terr != nil {
				return internalServerError("Error sending change email").WithInternalError(terr)
			}
		}

		if terr = models.NewAuditLogEntry(tx, instanceID, user, models.UserModifiedAction, nil); terr != nil {
			return internalServerError("Error recording audit log entry").WithInternalError(terr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, user)
}
