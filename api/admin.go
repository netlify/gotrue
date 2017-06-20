package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/models"
)

// adminUserParams are used to handle admin requests that relate to user accounts
// The User field is used for sub-user authentication and the others are used in the Update/Create endpoints
//
// To create a new user the request would look like:
//     {"email": "email@provider.com", "password": "password"}
//
// And to authenticate as another user as an administrator you would send:
//     {"user": {"email": "email@provider.com", "aud": "myaudience"}}
type adminUserParams struct {
	Role     string                 `json:"role"`
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Confirm  bool                   `json:"confirm"`
	Data     map[string]interface{} `json:"data"`
	User     struct {
		Aud   string `json:"aud"`
		Email string `json:"email"`
		ID    string `json:"_id"`
	} `json:"user"`
}

// Check the request to make sure the token is associated with an administrator
// Returns the admin user, the target user, the adminUserParams, audience name and a boolean designating whether or not the prior values are valid
func (api *API) checkAdmin(ctx context.Context, w http.ResponseWriter, r *http.Request, requireUser bool) (*models.User, *models.User, *adminUserParams, string, bool) {
	params := adminUserParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not decode admin user params: %v", err))
		return nil, nil, nil, "", false
	}

	// Find the administrative user
	adminUser, err := getUser(ctx, api.db)
	if err != nil {
		UnauthorizedError(w, "Invalid user")
		return nil, nil, nil, "", false
	}

	aud := api.requestAud(ctx, r)
	if params.User.Aud != "" {
		aud = params.User.Aud
	}

	// Make sure user is admin
	if !api.isAdmin(adminUser, aud) {
		UnauthorizedError(w, "Not allowed")
		return nil, nil, nil, aud, false
	}

	user, err := api.db.FindUserByEmailAndAudience(params.User.Email, params.User.Aud)
	if err != nil {
		if user, err = api.db.FindUserByID(params.User.ID); err != nil && requireUser {
			BadRequestError(w, fmt.Sprintf("Unable to find user by email: %s and id: %s in audience: %s", params.User.Email, params.User.ID, params.User.Aud))
			return nil, nil, nil, aud, false
		}
	}

	return adminUser, user, &params, aud, true
}

// adminUsers responds with a list of all users in a given audience
func (api *API) adminUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	adminUser, err := getUser(ctx, api.db)
	if err != nil {
		UnauthorizedError(w, "Invalid user")
		return
	}

	aud := api.requestAud(ctx, r)
	if !api.isAdmin(adminUser, aud) {
		UnauthorizedError(w, "Not allowed")
		return
	}

	users, err := api.db.FindUsersInAudience(aud)
	if err != nil {
		InternalServerError(w, err.Error())
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"aud":   aud,
	})
}

// adminUserGet returns information about a single user
func (api *API) adminUserGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, user, _, _, allowed := api.checkAdmin(ctx, w, r, true)
	if allowed {
		sendJSON(w, http.StatusOK, user)
	}
}

// adminUserUpdate updates a single user object
func (api *API) adminUserUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, user, params, _, allowed := api.checkAdmin(ctx, w, r, true)
	if !allowed {
		return
	}

	if params.Role != "" {
		user.SetRole(params.Role)
	}

	if params.Confirm {
		user.Confirm()
	}

	if params.Password != "" {
		user.EncryptPassword(params.Password)
	}

	if params.Email != "" {
		user.Email = params.Email
	}

	if err := api.db.UpdateUser(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error updating user %v", err))
		return
	}

	sendJSON(w, http.StatusOK, user)
}

// adminUserCreate creates a new user based on the provided data
func (api *API) adminUserCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, _, params, aud, allowed := api.checkAdmin(ctx, w, r, false)
	if !allowed {
		return
	}

	if err := mailer.ValidateEmail(params.Email); err != nil && !api.config.Testing {
		BadRequestError(w, fmt.Sprintf("Invalid email address: %s", params.Email))
		return
	}

	user, err := models.NewUser(params.Email, params.Password, aud, params.Data)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	if params.Role != "" {
		user.SetRole(params.Role)
	} else {
		user.SetRole(api.config.JWT.DefaultGroupName)
	}

	if params.Confirm {
		user.Confirm()
	}

	if err = api.db.CreateUser(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error creating new user: %v", err))
		return
	}

	sendJSON(w, http.StatusOK, user)
}

// adminUserDelete delete a user
func (api *API) adminUserDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, user, _, _, allowed := api.checkAdmin(ctx, w, r, true)
	if !allowed {
		return
	}

	if err := api.db.DeleteUser(user); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{})
}
