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
		user *models.User
		err  error
	)

	switch params.Type {
	case signupVerification:
		user, err = a.signupVerify(ctx, params)
	case recoveryVerification:
		user, err = a.recoverVerify(ctx, params)
	default:
		return unprocessableEntityError("Verify requires a verification type")
	}

	if err != nil {
		return err
	}

	if cookie != "" && config.Cookie.Duration > 0 {
		if err = a.setCookieToken(config, user, cookie == useSessionCookie, w); err != nil {
			return internalServerError("Failed to set JWT cookie", err)
		}
	}

	return a.sendRefreshToken(ctx, user, w)
}

func (a *API) signupVerify(ctx context.Context, params *VerifyParams) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)

	user, err := a.db.FindUserByConfirmationToken(params.Token)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(err.Error())
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	if user.EncryptedPassword == "" {
		if user.InvitedAt != nil {
			if params.Password == "" {
				return nil, unprocessableEntityError("Invited users must specify a password")
			}
			if err = user.EncryptPassword(params.Password); err != nil {
				return nil, internalServerError("Error encrypting password").WithInternalError(err)
			}
		}
	}

	if config.Webhook.HasEvent("signup") {
		if err := triggerHook(SignupEvent, user, instanceID, config); err != nil {
			return nil, err
		}
		a.db.UpdateUser(user)
	}

	user.Confirm()
	return user, nil
}

func (a *API) recoverVerify(ctx context.Context, params *VerifyParams) (*models.User, error) {
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)
	user, err := a.db.FindUserByRecoveryToken(params.Token)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError(err.Error())
		}
		return nil, internalServerError("Database error finding user").WithInternalError(err)
	}

	user.Recover()
	if !user.IsConfirmed() {
		if config.Webhook.HasEvent("signup") {
			if err := triggerHook(SignupEvent, user, instanceID, config); err != nil {
				return nil, err
			}
			a.db.UpdateUser(user)
		}
		user.Confirm()
	}
	return user, nil
}
