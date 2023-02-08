package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/go-chi/chi"
	"github.com/netlify/gotrue/models"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"github.com/tigrisdata/tigris-client-go/filter"
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
	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		return nil, badRequestError("user_id must be an UUID")
	}

	logEntrySetField(r, "user_id", userID)
	instanceID := getInstanceID(r.Context())

	u, err := models.FindUserByInstanceIDAndID(r.Context(), a.db, instanceID, userID)
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

	sortParams, err := sort(r, map[string]bool{models.CreatedAt: true}, []models.SortField{models.SortField{Name: models.CreatedAt, Dir: models.Descending}})
	if err != nil {
		return badRequestError("Bad Sort Parameters: %v", err)
	}

	filter := r.URL.Query().Get("filter")

	users, err := models.FindUsersInAudience(ctx, a.db, instanceID, aud, pageParams, sortParams, filter)
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
	ctx := r.Context()
	user := getUser(ctx)
	adminUser := getAdminUser(ctx)
	instanceID := getInstanceID(ctx)
	params, err := a.getAdminParams(r)
	if err != nil {
		return err
	}

	err = a.db.Tx(ctx, func(ctx context.Context) error {
		if params.Role != "" {
			if terr := user.SetRole(ctx, a.db, params.Role); terr != nil {
				return terr
			}
		}

		if params.Confirm {
			if terr := user.Confirm(ctx, a.db); terr != nil {
				return terr
			}
		}

		if params.Password != "" {
			if terr := user.UpdatePassword(ctx, a.db, params.Password); terr != nil {
				return terr
			}
		}

		if params.Email != "" {
			if terr := user.SetEmail(ctx, a.db, params.Email); terr != nil {
				return terr
			}
		}

		if params.AppMetaData != nil {
			if terr := user.UpdateAppMetaData(ctx, a.db, params.AppMetaData); terr != nil {
				return terr
			}
		}

		if params.UserMetaData != nil {
			if terr := user.UpdateUserMetaData(ctx, a.db, params.UserMetaData); terr != nil {
				return terr
			}
		}

		if terr := models.NewAuditLogEntry(ctx, a.db, instanceID, adminUser, models.UserModifiedAction, map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
		}); terr != nil {
			return terr
		}
		return nil
	})

	if err != nil {
		return internalServerError("Error updating user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserCreate creates a new user based on the provided data
func (a *API) adminUserCreate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	instanceID := getInstanceID(ctx)
	adminUser := getAdminUser(ctx)
	params, err := a.getAdminParams(r)
	if err != nil {
		return err
	}

	if err := a.validateEmail(ctx, params.Email); err != nil {
		return err
	}

	aud := a.requestAud(ctx, r)
	if params.Aud != "" {
		aud = params.Aud
	}

	if exists, err := models.IsDuplicatedEmail(ctx, a.db, instanceID, params.Email, aud); err != nil {
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
	err = a.db.Tx(ctx, func(ctx context.Context) error {
		if terr := models.NewAuditLogEntry(ctx, a.db, instanceID, adminUser, models.UserSignedUpAction, map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
		}); terr != nil {
			return terr
		}

		if terr := user.BeforeCreate(); terr != nil {
			return terr
		}

		_, terr := tigris.GetCollection[models.User](a.db).Insert(ctx, user)
		if terr != nil {
			return terr
		}

		role := config.JWT.DefaultGroupName
		if params.Role != "" {
			role = params.Role
		}
		if terr := user.SetRole(ctx, a.db, role); terr != nil {
			return terr
		}

		if params.Confirm {
			if terr := user.Confirm(ctx, a.db); terr != nil {
				return terr
			}
		}

		return nil
	})

	if err != nil {
		return internalServerError("Database error creating new user").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, user)
}

// adminUserDelete delete a user
func (a *API) adminUserDelete(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := getUser(ctx)
	instanceID := getInstanceID(ctx)
	adminUser := getAdminUser(ctx)

	err := a.db.Tx(ctx, func(ctx context.Context) error {
		if terr := models.NewAuditLogEntry(ctx, a.db, instanceID, adminUser, models.UserDeletedAction, map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
		}); terr != nil {
			return internalServerError("Error recording audit log entry").WithInternalError(terr)
		}

		_, terr := tigris.GetCollection[models.User](a.db).Delete(ctx, filter.EqUUID("id", user.ID))
		if terr != nil {
			return internalServerError("Database error deleting user").WithInternalError(terr)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, map[string]interface{}{})
}
