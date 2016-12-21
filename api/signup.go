package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/netlify-auth/models"
)

// SignupParams are the parameters the Signup endpoint accepts
type SignupParams struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Data     map[string]interface{} `json:"data"`
}

// Signup is the endpoint for registering a new user
func (a *API) Signup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var user *models.User
	params := &SignupParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		BadRequestError(w, fmt.Sprintf("Could not read Signup params: %v", err))
		return
	}

	if params.Email == "" || params.Password == "" {
		UnprocessableEntity(w, fmt.Sprintf("Signup requires a valid email and password"))
		return
	}

	existingUser := &models.User{}

	tx := a.db.Begin()
	if result := tx.First(existingUser, "email = ?", params.Email); result.Error != nil {
		if result.RecordNotFound() {
			user, err = models.CreateUser(tx, params.Email, params.Password)
			if err != nil {
				InternalServerError(w, fmt.Sprintf("Error creating user: %v", err))
				return
			}
			fmt.Printf("Created new user: %v", user)
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
			return
		}
	} else {
		if !existingUser.ConfirmedAt.IsZero() {
			UnprocessableEntity(w, fmt.Sprintf("A user with this email address has already been registered"))
			return
		}
		existingUser.GenerateConfirmationToken()

		tx.Save(existingUser)
		user = existingUser
	}

	users := []*models.User{}
	var userCount int64
	if result := tx.Where("id != ?", user.ID).Find(&users).Count(&userCount); result.Error != nil {
		InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		return
	}

	if params.Data != nil {
		user.UpdateUserMetaData(tx, &params.Data)
	}

	if userCount == 0 {
		user.UpdateAppMetaData(tx, &map[string]interface{}{"roles": []string{a.config.JWT.AdminGroupName}})
	}

	if err := a.mailer.ConfirmationMail(user); err != nil {
		InternalServerError(w, fmt.Sprintf("Error sending confirmation mail: %v", err))
		return
	}

	tx.Commit()
	sendJSON(w, 200, user)
}
