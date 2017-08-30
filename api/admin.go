package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/netlify/gotrue/models"
)

type adminUserParams struct {
	Aud          string                 `json:"aud"`
	Role         string                 `json:"role"`
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	Confirm      bool                   `json:"confirm"`
	UserMetaData map[string]interface{} `json:"user_metadata"`
	AppMetaData  map[string]interface{} `json:"app_metadata"`
}

func (a *API) loadUser(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	userID := chi.URLParam(r, "user_id")
	logEntrySetField(r, "user_id", userID)
	instanceID := getInstanceID(r.Context())

	u, err := a.db.FindUserByInstanceIDAndID(instanceID, userID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError("User not found")
		}
		return nil, internalServerError("Database error loading user").WithInternalError(err)
	}

	return withUser(r.Context(), u), nil
}

func (a *API) getAdminParams(r *http.Request) (*adminUserParams, error) {
	params := adminUserParams{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		return nil, badRequestError("Could not decode admin user params: %v", err)
	}
	return &params, nil
}

// adminUsers responds with a list of all users in a given audience
func (a *API) adminUsers(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(ctx)
	aud := a.requestAud(ctx, r)

	pageParams, err := paginate(r)
	if err != nil {
		return badRequestError("Bad Pagination Parameters: %v", err)
	}

	sortParams, err := sort(r, map[string]bool{"created_at": true}, []models.SortField{models.SortField{Name: "created_at", Dir: models.Descending}})
	if err != nil {
		return badRequestError("Bad Sort Parameters: %v", err)
	}

	users, err := a.db.FindUsersInAudience(instanceID, aud, pageParams, sortParams)
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
func (a *API) adminUserGet(w http.ResponseWriter, r *http.Request) error {
	user := getUser(r.Context())

	return sendJSON(w, http.StatusOK, user)
}

// adminUserUpdate updates a single user object
func (a *API) adminUserUpdate(w http.ResponseWriter, r *http.Request) error {
	user := getUser(r.Context())
	params, err := a.getAdminParams(r)
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

	if params.AppMetaData != nil {
		user.UpdateAppMetaData(params.AppMetaData)
	}

	if params.UserMetaData != nil {
		user.UpdateUserMetaData(params.UserMetaData)
	}

	if err := a.db.UpdateUser(user); err != nil {
		return internalServerError("Error updating user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserCreate creates a new user based on the provided data
func (a *API) adminUserCreate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(ctx)
	params, err := a.getAdminParams(r)
	if err != nil {
		return err
	}

	mailer := getMailer(ctx)
	if err := mailer.ValidateEmail(params.Email); err != nil {
		return badRequestError("Invalid email address: %s", params.Email).WithInternalError(err)
	}

	aud := a.requestAud(ctx, r)
	if params.Aud != "" {
		aud = params.Aud
	}

	if exists, err := a.db.IsDuplicatedEmail(instanceID, params.Email, aud); err != nil {
		return internalServerError("Database error checking email").WithInternalError(err)
	} else if exists {
		return unprocessableEntityError("Email address already registered by another user")
	}

	user, err := models.NewUser(instanceID, params.Email, params.Password, aud, params.UserMetaData)
	if err != nil {
		return internalServerError("Error creating user").WithInternalError(err)
	}
	if user.AppMetaData == nil {
		user.AppMetaData = make(map[string]interface{})
	}
	user.AppMetaData["provider"] = "email"

	config := a.getConfig(ctx)
	if params.Role != "" {
		user.SetRole(params.Role)
	} else {
		user.SetRole(config.JWT.DefaultGroupName)
	}

	if params.Confirm {
		user.Confirm()
	}

	if err = a.db.CreateUser(user); err != nil {
		return internalServerError("Database error creating new user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserDelete delete a user
func (a *API) adminUserDelete(w http.ResponseWriter, r *http.Request) error {
	user := getUser(r.Context())

	if err := a.db.DeleteUser(user); err != nil {
		return internalServerError("Database error deleting user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, map[string]interface{}{})
}
