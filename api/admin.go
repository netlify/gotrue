package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/models"
)

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

func (api *API) checkAdmin(ctx context.Context, w http.ResponseWriter, r *http.Request) (*models.User, *models.User, *adminUserParams, bool) {
	// Get User associated with incoming request
	params := adminUserParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not decode admin user params: %v", err))
		return nil, nil, nil, false
	}

	adminUser, err := getUser(ctx, api.db)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return nil, nil, nil, false
	}

	// Make sure user is admin
	if !api.isAdmin(adminUser, api.requestAud(ctx, r)) {
		UnauthorizedError(w, "Not allowed")
		return nil, nil, nil, false
	}

	user, err := api.db.FindUserByEmailAndAudience(params.User.Email, params.User.Aud)
	if err != nil {
		fmt.Println("NO USER")
		if user, err = api.db.FindUserByID(params.User.ID); err != nil {
			if models.IsNotFoundError(err) {
				NotFoundError(w, err.Error())
			} else {
				InternalServerError(w, err.Error())
			}
			return nil, nil, nil, false
		}
	}

	return adminUser, user, &params, true

}

func (api *API) adminUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	adminUser, err := getUser(ctx, api.db)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	aud := api.requestAud(ctx, r)
	if !api.isAdmin(adminUser, aud) {
		UnauthorizedError(w, "Not allowed")
		return
	}

	users := api.db.FindUsersInAudience(aud)
	sendJSON(w, 200, map[string]interface{}{
		"users": users,
		"aud":   aud,
	})
}

func (api *API) adminUserGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, user, _, allowed := api.checkAdmin(ctx, w, r)
	if allowed {
		sendJSON(w, 200, user)
	}
}

func (api *API) adminUserUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, user, params, allowed := api.checkAdmin(ctx, w, r)
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

	sendJSON(w, 200, user)
}

func (api *API) adminUserCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := adminUserParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(&params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not decode admin user params: %v", err))
		return
	}

	adminUser, err := getUser(ctx, api.db)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	aud := api.requestAud(ctx, r)
	if !api.isAdmin(adminUser, aud) {
		UnauthorizedError(w, "Not allowed")
		return
	}

	if err = mailer.ValidateEmail(params.Email); err != nil && !api.config.Testing {
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

	sendJSON(w, 200, user)
}

func (api *API) adminUserDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, user, _, allowed := api.checkAdmin(ctx, w, r)
	if !allowed {
		return
	}

	if err := api.db.DeleteUser(user); err != nil {
		InternalServerError(w, err.Error())
		return
	}

	sendJSON(w, 200, map[string]interface{}{})
}
