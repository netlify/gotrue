package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/netlify/gotrue/models"
)

// SignupParams are the parameters the Signup endpoint accepts
type SignupParams struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Data     map[string]interface{} `json:"data"`
	Provider string                 `json:"provider"`
	Code     string                 `json:"code"`
}

func (a *API) signupExternalProvider(ctx context.Context, w http.ResponseWriter, r *http.Request, params *SignupParams) {
	provider, err := a.Provider(params.Provider)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Unsupported provider: %+v", err))
		return
	}

	tok, err := provider.GetOAuthToken(ctx, params.Code)
	if err != nil {
		InternalServerError(w, fmt.Sprintf("Unable to exchange external code: %+v", err))
		return
	}

	aud := a.requestAud(ctx, r)
	params.Email, err = provider.GetUserEmail(ctx, tok)
	if err != nil {
		InternalServerError(w, fmt.Sprintf("Error getting user email: %+v", err))
		return
	}

	if exists, err := a.db.IsDuplicatedEmail(params.Email, aud); exists {
		BadRequestError(w, "User already exists")
		return
	} else if err != nil {
		InternalServerError(w, fmt.Sprintf("Error checking for duplicate users: %+v", err))
		return
	}

	user, err := a.signupNewUser(params, aud)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	if a.config.Mailer.Autoconfirm {
		user.Confirm()
	} else if user.ConfirmationSentAt.Add(time.Minute * 15).Before(time.Now()) {
		if err := a.mailer.ConfirmationMail(user); err != nil {
			InternalServerError(w, fmt.Sprintf("Error sending confirmation mail: %v", err))
			return
		}
	}

	user.SetRole(a.config.JWT.DefaultGroupName)
	if err := a.db.UpdateUser(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error updating user in database: %v", err))
		return

	}

	sendJSON(w, 200, user)
}

// Signup is the endpoint for registering a new user
func (a *API) Signup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := &SignupParams{}

	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read Signup params: %v", err))
		return
	}

	if params.Provider != "" && params.Code != "" {
		a.signupExternalProvider(ctx, w, r, params)
		return
	}

	if params.Email == "" || params.Password == "" {
		UnprocessableEntity(w, "Signup requires a valid email and password")
		return
	}

	aud := a.requestAud(ctx, r)

	user, err := a.db.FindUserByEmailAndAudience(params.Email, aud)
	if err != nil {
		if !models.IsNotFoundError(err) {
			InternalServerError(w, err.Error())
			return
		}

		user, err = a.signupNewUser(params, aud)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}
	} else {
		user, err = a.confirmUser(user, params)
		if err != nil {
			InternalServerError(w, err.Error())
			return
		}
	}

	if err = a.mailer.ValidateEmail(params.Email); err != nil {
		UnprocessableEntity(w, "Unable to validate email address: "+err.Error())
		return
	}

	if a.config.Mailer.Autoconfirm {
		user.Confirm()
	} else if user.ConfirmationSentAt.Add(time.Minute * 15).Before(time.Now()) {
		if err := a.mailer.ConfirmationMail(user); err != nil {
			InternalServerError(w, fmt.Sprintf("Error sending confirmation mail: %v", err))
			return
		}
	}

	user.SetRole(a.config.JWT.DefaultGroupName)
	a.db.UpdateUser(user)

	sendJSON(w, 200, user)
}

func (a *API) signupNewUser(params *SignupParams, aud string) (*models.User, error) {
	user, err := models.NewUser(params.Email, params.Password, aud, params.Data)
	if err != nil {
		return nil, err
	}

	if err := a.db.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (a *API) confirmUser(user *models.User, params *SignupParams) (*models.User, error) {
	if user.IsRegistered() {
		return nil, errors.New("A user with this email address has already been registered")
	}

	user.UpdateUserMetaData(params.Data)
	user.GenerateConfirmationToken()

	if err := a.db.UpdateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}
