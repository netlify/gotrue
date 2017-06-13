package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/models"
)

type adminRoleParams struct {
	Role  string `json:"role"`
	Aud   string `json:"aud"`
	Email string `json:"email"`
}

func (api *API) adminUserRoleGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get User associated with incoming request
	params := adminRoleParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read Admin Set Role params: %v", err))
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

	// Make sure user is admin
	if !api.isAdmin(adminUser, params.Aud) {
		UnauthorizedError(w, "Not allowed")
		return
	}

	user, err := api.db.FindUserByEmailAndAudience(params.Email, params.Aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	sendJSON(w, 200, map[string]interface{}{
		"aud":         user.Aud,
		"role":        user.Role,
		"super_admin": user.IsSuperAdmin,
	})
}

func (api *API) adminUserRoleSet(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get User associated with incoming request
	params := &adminRoleParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read Admin Set Role params: %v", err))
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

	// Make sure user is admin
	if !api.isAdmin(adminUser, params.Aud) {
		UnauthorizedError(w, "Not allowed")
		return
	}

	user, err := api.db.FindUserByEmailAndAudience(params.Email, params.Aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			NotFoundError(w, err.Error())
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	user.SetRole(params.Role)

	if err = api.db.UpdateUser(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error setting role: %v", err))
		return
	}

	sendJSON(w, 200, map[string]interface{}{
		"aud":  user.Aud,
		"role": user.Role,
	})
}
