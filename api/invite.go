package api

import (
	"encoding/json"
	"net/http"

	"github.com/netlify/gotrue/models"
	"context"
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
	adminUser := getAdminUser(ctx)
	params := &InviteParams{}

	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read Invite params: %v", err)
	}

	if err := a.validateEmail(ctx, params.Email); err != nil {
		return err
	}

	aud := a.requestAud(ctx, r)
	user, err := models.FindUserByEmailAndAudience(ctx, a.db, instanceID, params.Email, aud)
	if err != nil && !models.IsNotFoundError(err) {
		return internalServerError("Database error finding user").WithInternalError(err)
	}
	if user != nil {
		return unprocessableEntityError("Email address already registered by another user")
	}

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		signupParams := SignupParams{
			Email:    params.Email,
			Data:     params.Data,
			Aud:      aud,
			Provider: "email",
		}
		user, err = a.signupNewUser(ctx, &signupParams)
		if err != nil {
			return err
		}

		if terr := models.NewAuditLogEntry(r.Context(), a.db, instanceID, adminUser, models.UserInvitedAction, map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
		}); terr != nil {
			return terr
		}

		mailer := a.Mailer(ctx)
		referrer := a.getReferrer(r)
		if err := sendInvite(ctx, a.db, user, mailer, referrer); err != nil {
			return internalServerError("Error inviting user").WithInternalError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, user)
}
