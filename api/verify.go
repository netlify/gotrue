package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/models"
)

const (
	signupVerification   = "signup"
	recoveryVerification = "recovery"
)

// VerifyParams are the parameters the Verify endpoint accepts
type VerifyParams struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Verify exchanges a confirmation or recovery token to a refresh token
func (a *API) Verify(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := &VerifyParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	if err := jsonDecoder.Decode(params); err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read verification params: %v", err))
		return
	}

	if params.Token == "" {
		UnprocessableEntity(w, "Verify requires a token")
		return
	}

	var (
		user       *models.User
		err        error
		verifyFunc func(user *models.User)
	)

	switch params.Type {
	case signupVerification:
		user, err = a.db.FindUserByConfirmationToken(params.Token)
		verifyFunc = func(user *models.User) { user.Confirm() }
	case recoveryVerification:
		user, err = a.db.FindUserByRecoveryToken(params.Token)
		verifyFunc = func(user *models.User) { user.Recover() }
	default:
		UnprocessableEntity(w, "Verify requires a verification type")
		return
	}

	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	verifyFunc(user)
	a.issueRefreshToken(user, w)
}
