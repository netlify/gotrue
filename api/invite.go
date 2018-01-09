package api

import (
	"encoding/json"
	"net/http"

	"github.com/netlify/gotrue/models"
)

// InviteParams are the parameters the Signup endpoint accepts
type InviteParams struct {
	Email string                 `json:"email"`
	Data  map[string]interface{} `json:"data"`
}

// Invite is the endpoint for inviting a new user
func (a *API) Invite(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(ctx)
	params := &InviteParams{}

	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read Invite params: %v", err)
	}

	if params.Email == "" {
		return unprocessableEntityError("Invite requires a valid email")
	}
	mailer := a.Mailer(ctx)
	if err = mailer.ValidateEmail(params.Email); err != nil {
		return unprocessableEntityError("Unable to validate email address: " + err.Error())
	}

	aud := a.requestAud(ctx, r)
	user, err := a.db.FindUserByEmailAndAudience(instanceID, params.Email, aud)
	if err == nil {
		return unprocessableEntityError("Email address already registered by another user")
	}
	if !models.IsNotFoundError(err) {
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	signupParams := SignupParams{
		Email:    params.Email,
		Data:     params.Data,
		Provider: "email",
	}

	user, err = a.signupNewUser(ctx, &signupParams, aud)
	if err != nil {
		return err
	}

	if err := a.sendInvite(user, mailer); err != nil {
		return internalServerError("Error inviting user").WithInternalError(err)
	}
	return sendJSON(w, http.StatusOK, user)
}
