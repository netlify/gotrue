package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/netlify/netlify-auth/models"
)

// UserUpdateParams parameters for updating a user
type UserUpdateParams struct {
	Email            string                 `json:"email"`
	Password         string                 `json:"password"`
	EmailChangeToken string                 `json:"email_change_token"`
	Data             map[string]interface{} `json:"data"`
	AppData          map[string]interface{} `json:"app_metadata,omitempty"`
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

	if params.Email != "" {
		existingUser := &models.User{}
		result := tx.First(existingUser, "id != ? and email = ?", user.ID, params.Email)
		if result.RecordNotFound() {
			user.GenerateEmailChange(params.Email)
			a.mailer.EmailChangeMail(user)
		} else {
			tx.Rollback()
			if result.Error != nil {
				InternalServerError(w, fmt.Sprintf("Error during database query:%v", result.Error))
			} else {
				UnprocessableEntity(w, "Email address already registered by another user")
			}
		}
	}

	log.Printf("Checkign params fortoken %v", params)
	if params.EmailChangeToken != "" {
		log.Printf("Got change token %v", params.EmailChangeToken)
		if params.EmailChangeToken == user.EmailChangeToken {
			log.Printf("Confirm email change")
			user.ConfirmEmailChange()
		} else {
			tx.Rollback()
			UnauthorizedError(w, "Email Change Token didn't match token on file")
			return
		}
	}

	if params.Password != "" {
		if err = user.EncryptPassword(params.Password); err != nil {
			tx.Rollback()
			InternalServerError(w, fmt.Sprintf("Error during password encryption: %v", err))
		}
	}

	if params.Data != nil {
		if err = user.UpdateUserMetaData(tx, &params.Data); err != nil {
			tx.Rollback()
			InternalServerError(w, err.Error())
			return
		}
	}

	if params.AppData != nil {
		if !user.HasRole(a.config.JWT.AdminGroupName) {
			tx.Rollback()
			UnauthorizedError(w, "Updating app_metadata requires admin privileges")
			return
		}
		if err = user.UpdateAppMetaData(tx, &params.AppData); err != nil {
			tx.Rollback()
			InternalServerError(w, err.Error())
			return
		}
	}

	tx.Model(user).Update(user)
	tx.Commit()
	sendJSON(w, 200, user)
}
