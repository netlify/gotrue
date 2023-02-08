package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/netlify/gotrue/models"
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

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		var terr error
		switch params.Type {
		case signupVerification:
			user, terr = a.signupVerify(ctx, params)
		case recoveryVerification:
			user, terr = a.recoverVerify(ctx, params)
		default:
			return unprocessableEntityError("Verify requires a verification type")
		}

		if terr != nil {
			return terr
		}

		token, terr = a.issueRefreshToken(ctx, user)
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

func (a *API) signupVerify(ctx context.Context, params *VerifyParams) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)

	user, err := models.FindUserByConfirmationToken(ctx, a.db, params.Token)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(err.Error())
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		var terr error
		if user.EncryptedPassword == "" {
			if user.InvitedAt != nil {
				if params.Password == "" {
					return unprocessableEntityError("Invited users must specify a password")
				}
				if terr = user.UpdatePassword(ctx, a.db, params.Password); terr != nil {
					return internalServerError("Error storing password").WithInternalError(terr)
				}
			}
		}

		if terr = models.NewAuditLogEntry(ctx, a.db, instanceID, user, models.UserSignedUpAction, nil); terr != nil {
			return terr
		}

		if terr = triggerEventHooks(ctx, a.db, SignupEvent, user, instanceID, config); terr != nil {
			return terr
		}

		if terr = user.Confirm(ctx, a.db); terr != nil {
			return internalServerError("Error confirming user").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a *API) recoverVerify(ctx context.Context, params *VerifyParams) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)
	user, err := models.FindUserByRecoveryToken(ctx, a.db, params.Token)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(err.Error())
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		var terr error
		if terr = user.Recover(ctx, a.db); terr != nil {
			return terr
		}
		if !user.IsConfirmed() {
			if terr = models.NewAuditLogEntry(ctx, a.db, instanceID, user, models.UserSignedUpAction, nil); terr != nil {
				return terr
			}

			if terr = triggerEventHooks(ctx, a.db, SignupEvent, user, instanceID, config); terr != nil {
				return terr
			}
			if terr = user.Confirm(ctx, a.db); terr != nil {
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
