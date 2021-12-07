package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
)

const (
	signupVerification   = "signup"
	recoveryVerification = "recovery"
)

// VerifyParams are the parameters the Verify endpoint accepts
type VerifyParams struct {
	Type     string `json:"type"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

// Verify exchanges a confirmation or recovery token to a refresh token
func (a *API) Verify(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)

	params := &VerifyParams{}
	cookie := r.Header.Get(useCookieHeader)
	jsonDecoder := json.NewDecoder(r.Body)
	if err := jsonDecoder.Decode(params); err != nil {
		return badRequestError("Could not read verification params: %v", err)
	}

	if params.Token == "" {
		return unprocessableEntityError("Verify requires a token")
	}

	var (
		user  *models.User
		err   error
		token *AccessTokenResponse
	)

	err = a.db.Transaction(func(tx *storage.Connection) error {
		var terr error
		switch params.Type {
		case signupVerification:
			user, terr = a.signupVerify(ctx, tx, params)
		case recoveryVerification:
			user, terr = a.recoverVerify(ctx, tx, params)
		default:
			return unprocessableEntityError("Verify requires a verification type")
		}

		if terr != nil {
			return terr
		}

		token, terr = a.issueRefreshToken(ctx, tx, user)
		if terr != nil {
			return terr
		}

		if cookie != "" && config.Cookie.Duration > 0 {
			if terr = a.setCookieToken(config, token.Token, cookie == useSessionCookie, w); terr != nil {
				return internalServerError("Failed to set JWT cookie. %s", terr)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, token)
}

func (a *API) signupVerify(ctx context.Context, conn *storage.Connection, params *VerifyParams) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)

	user, err := models.FindUserByConfirmationToken(conn, params.Token)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(err.Error())
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	err = conn.Transaction(func(tx *storage.Connection) error {
		var terr error
		if user.EncryptedPassword == "" {
			if user.InvitedAt != nil {
				if params.Password == "" {
					return unprocessableEntityError("Invited users must specify a password")
				}
				if terr = user.UpdatePassword(tx, params.Password); terr != nil {
					return internalServerError("Error storing password").WithInternalError(terr)
				}
			}
		}

		if terr = models.NewAuditLogEntry(tx, instanceID, user, models.UserSignedUpAction, nil); terr != nil {
			return terr
		}

		if terr = triggerEventHooks(ctx, tx, SignupEvent, user, instanceID, config); terr != nil {
			return terr
		}

		if terr = user.Confirm(tx); terr != nil {
			return internalServerError("Error confirming user").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a *API) recoverVerify(ctx context.Context, conn *storage.Connection, params *VerifyParams) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)
	user, err := models.FindUserByRecoveryToken(conn, params.Token)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(err.Error())
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	err = conn.Transaction(func(tx *storage.Connection) error {
		var terr error
		if terr = user.Recover(tx); terr != nil {
			return terr
		}
		if !user.IsConfirmed() {
			if terr = models.NewAuditLogEntry(tx, instanceID, user, models.UserSignedUpAction, nil); terr != nil {
				return terr
			}

			if terr = triggerEventHooks(ctx, tx, SignupEvent, user, instanceID, config); terr != nil {
				return terr
			}
			if terr = user.Confirm(tx); terr != nil {
				return terr
			}
		}
		return nil
	})

	if err != nil {
		return nil, internalServerError("Database error updating user").WithInternalError(err)
	}
	return user, nil
}
