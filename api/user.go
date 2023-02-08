package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/netlify/gotrue/models"
	"github.com/google/uuid"
	"context"
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
func (a *API) UserGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		return badRequestError("Could not read claims")
	}

	userID, err := uuid.Parse(GetUserIdFromSubject(claims.Subject))
	if err != nil {
		return badRequestError("Could not read User ID claim")
	}

	aud := a.requestAud(ctx, r)
	if aud != claims.Audience {
		return badRequestError("Token audience doesn't match request audience")
	}

	user, err := models.FindUserByID(r.Context(), a.db, userID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

func GetUserIdFromSubject(subject string) string {
	return strings.Replace(subject, "gt|", "", 1)
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
	userID, err := uuid.Parse(GetUserIdFromSubject(claims.Subject))
	if err != nil {
		return badRequestError("Could not read User ID claim")
	}

	user, err := models.FindUserByID(r.Context(), a.db, userID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	log := getLogEntry(r)
	log.Debugf("Checking params for token %v", params)

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		var terr error
		if params.Password != "" {
			if terr = user.UpdatePassword(ctx, a.db, params.Password); terr != nil {
				return internalServerError("Error during password storage").WithInternalError(terr)
			}
		}

		if params.Data != nil {
			if terr = user.UpdateUserMetaData(ctx, a.db, params.Data); terr != nil {
				return internalServerError("Error updating user").WithInternalError(terr)
			}
		}

		if params.AppData != nil {
			if !a.isAdmin(ctx, user, config.JWT.Aud) {
				return unauthorizedError("Updating app_metadata requires admin privileges")
			}

			if terr = user.UpdateAppMetaData(ctx, a.db, params.AppData); terr != nil {
				return internalServerError("Error updating user").WithInternalError(terr)
			}
		}

		if params.EmailChangeToken != "" {
			log.Debugf("Got change token %v", params.EmailChangeToken)

			if params.EmailChangeToken != user.EmailChangeToken {
				return unauthorizedError("Email Change Token didn't match token on file")
			}

			if terr = user.ConfirmEmailChange(ctx, a.db); terr != nil {
				return internalServerError("Error updating user").WithInternalError(terr)
			}
		} else if params.Email != "" && params.Email != user.Email {
			if terr = a.validateEmail(ctx, params.Email); terr != nil {
				return terr
			}

			var exists bool
			if exists, terr = models.IsDuplicatedEmail(ctx, a.db, instanceID, params.Email, user.Aud); terr != nil {
				return internalServerError("Database error checking email").WithInternalError(terr)
			} else if exists {
				return unprocessableEntityError("Email address already registered by another user")
			}

			mailer := a.Mailer(ctx)
			referrer := a.getReferrer(r)
			if terr = a.sendEmailChange(ctx, a.db, user, mailer, params.Email, referrer); terr != nil {
				return internalServerError("Error sending change email").WithInternalError(terr)
			}
		}

		if terr = models.NewAuditLogEntry(ctx, a.db, instanceID, user, models.UserModifiedAction, nil); terr != nil {
			return internalServerError("Error recording audit log entry").WithInternalError(terr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, user)
}
