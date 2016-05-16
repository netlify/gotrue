package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

// UserUpdateParams parameters for updating a user
type UserUpdateParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserGet returns a user
func (a *API) UserGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	user := &models.User{}
	if result := a.db.First(user, "id = ?", token.Claims["id"]); result.Error != nil {
		if result.RecordNotFound() {
			NotFoundError(w, "No user found for this token")
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	sendJSON(w, 200, user)
}

// UserUpdate updates fields on a user
func (a *API) UserUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	params := &UserUpdateParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read User Update params: %v", err))
		return
	}

	if params.Email == "" && params.Password == "" {
		UnprocessableEntity(w, fmt.Sprintf("Update requires either an email or a password"))
		return
	}

	tx := a.db.Begin()

	user := &models.User{}
	if result := tx.First(user, "id = ?", token.Claims["id"]); result.Error != nil {
		tx.Rollback()
		if result.RecordNotFound() {
			NotFoundError(w, "No user found for this token")
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	// TODO: we should probably do an email verification for this?
	if params.Email != "" {
		existingUser := &models.User{}
		result := tx.First(existingUser, "id != ? and email = ?", user.ID, params.Email)
		if result.RecordNotFound() {
			user.Email = params.Email
		} else {
			tx.Rollback()
			if result.Error != nil {
				InternalServerError(w, fmt.Sprintf("Error during database query:%v", result.Error))
			} else {
				UnprocessableEntity(w, "Email address already registered by another user")
			}
		}
	}

	if params.Password != "" {
		if err := user.UpdatePassword(params.Password); err != nil {
			tx.Rollback()
			InternalServerError(w, fmt.Sprintf("Error during password encryption: %v", err))
		}
	}

	tx.Save(user)
	tx.Commit()
	sendJSON(w, 200, user)
}
