package api

import (
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
	Type  string `json:"type"`
	Token string `json:"token"`
}

// Verify exchanges a confirmation or recovery token to a refresh token
func (a *API) Verify(w http.ResponseWriter, r *http.Request) error {
	params := &VerifyParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	if err := jsonDecoder.Decode(params); err != nil {
		return badRequestError("Could not read verification params: %v", err)
	}

	if params.Token == "" {
		return unprocessableEntityError("Verify requires a token")
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
		return unprocessableEntityError("Verify requires a verification type")
	}

	if err != nil {
		if models.IsNotFoundError(err) {
			return notFoundError(err.Error())
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	verifyFunc(user)
	return a.issueRefreshToken(user, w)
}
