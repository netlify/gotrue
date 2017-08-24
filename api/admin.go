package api

import (
	"encoding/json"
	"net/http"

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
type adminTargetUser struct {
	Aud   string `json:"aud"`
	Email string `json:"email"`
	ID    string `json:"id"`
}

type adminUserParams struct {
	Role     string                 `json:"role"`
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Confirm  bool                   `json:"confirm"`
	Data     map[string]interface{} `json:"data"`
	User     adminTargetUser        `json:"user"`
}

func (api *API) getAdminParams(r *http.Request) (*adminUserParams, error) {
	params := adminUserParams{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		return nil, badRequestError("Could not decode admin user params: %v", err)
	}
	return &params, nil
}

// Returns the the target user
func (api *API) getAdminTargetUser(instanceID string, params *adminUserParams) (*models.User, error) {
	user, err := api.db.FindUserByEmailAndAudience(instanceID, params.User.Email, params.User.Aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			if user, err = api.db.FindUserByID(params.User.ID); err != nil {
				if models.IsNotFoundError(err) {
					return nil, badRequestError("Unable to find user by email: %s and id: %s in audience: %s", params.User.Email, params.User.ID, params.User.Aud)
				}
				return nil, internalServerError("Database error finding user").WithInternalError(err)
			}
		} else {
			return nil, internalServerError("Database error finding user").WithInternalError(err)
		}
	}

	return user, nil
}

// adminUsers responds with a list of all users in a given audience
func (api *API) adminUsers(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(ctx)
	aud := api.requestAud(ctx, r)

	pageParams, err := paginate(r)
	if err != nil {
		return badRequestError("Bad Pagination Parameters: %v", err)
	}

	users, err := api.db.FindUsersInAudience(instanceID, aud, pageParams)
	if err != nil {
		return internalServerError("Database error finding users").WithInternalError(err)
	}
	addPaginationHeaders(w, r, pageParams)

	return sendJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"aud":   aud,
	})
}

// adminUserGet returns information about a single user
func (api *API) adminUserGet(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(r.Context())

	aud := r.FormValue("aud")
	if aud == "" {
		aud = api.requestAud(ctx, r)
	}

	params := &adminUserParams{
		User: adminTargetUser{
			ID:    r.FormValue("id"),
			Email: r.FormValue("email"),
			Aud:   aud,
		},
	}
	user, err := api.getAdminTargetUser(instanceID, params)
	if err != nil {
		return err
	}
	return sendJSON(w, http.StatusOK, user)
}

// adminUserUpdate updates a single user object
func (api *API) adminUserUpdate(w http.ResponseWriter, r *http.Request) error {
	instanceID := getInstanceID(r.Context())
	params, err := api.getAdminParams(r)
	if err != nil {
		return err
	}
	user, err := api.getAdminTargetUser(instanceID, params)
	if err != nil {
		return err
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
		return internalServerError("Error updating user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserCreate creates a new user based on the provided data
func (api *API) adminUserCreate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(ctx)
	params, err := api.getAdminParams(r)
	if err != nil {
		return err
	}

	mailer := getMailer(ctx)
	if err := mailer.ValidateEmail(params.Email); err != nil {
		return badRequestError("Invalid email address: %s", params.Email).WithInternalError(err)
	}

	aud := api.requestAud(ctx, r)
	if params.User.Aud != "" {
		aud = params.User.Aud
	}

	if exists, err := api.db.IsDuplicatedEmail(instanceID, params.Email, aud); err != nil {
		return internalServerError("Database error checking email").WithInternalError(err)
	} else if exists {
		return unprocessableEntityError("Email address already registered by another user")
	}

	user, err := models.NewUser(instanceID, params.Email, params.Password, aud, params.Data)
	if err != nil {
		return internalServerError("Error creating user").WithInternalError(err)
	}

	config := getConfig(ctx)
	if params.Role != "" {
		user.SetRole(params.Role)
	} else {
		user.SetRole(config.JWT.DefaultGroupName)
	}

	if params.Confirm {
		user.Confirm()
	}

	if err = api.db.CreateUser(user); err != nil {
		return internalServerError("Database error creating new user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserDelete delete a user
func (api *API) adminUserDelete(w http.ResponseWriter, r *http.Request) error {
	instanceID := getInstanceID(r.Context())
	params, err := api.getAdminParams(r)
	if err != nil {
		return err
	}
	user, err := api.getAdminTargetUser(instanceID, params)
	if err != nil {
		return err
	}

	if err := api.db.DeleteUser(user); err != nil {
		return internalServerError("Database error deleting user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, map[string]interface{}{})
}
