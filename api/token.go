package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

// AccessTokenResponse represents an OAuth2 success response
type AccessTokenResponse struct {
	Token        string `json:"access_token"`
	TokenType    string `json:"token_type"` // Bearer
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// Token is the endpoint for OAuth access token requests
func (a *API) Token(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	grantType := r.FormValue("grant_type")

	switch grantType {
	case "password":
		a.ResourceOwnerPasswordGrant(ctx, w, r)
	case "refresh_token":
		a.RefreshTokenGrant(ctx, w, r)
	default:
		sendJSON(w, 400, &OAuthError{Error: "unsupported_grant_type"})
	}
}

// ResourceOwnerPasswordGrant implements the password grant type flow
func (a *API) ResourceOwnerPasswordGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user := &models.User{}
	if result := a.db.First(user, "email = ?", username); result.Error != nil {
		if result.RecordNotFound() {
			sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "No user found with this email"})
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	if user.ConfirmedAt.IsZero() {
		sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Email not confirmed"})
		return
	}

	if user.Authenticate(password) {
		tx := a.db.Begin()

		user.LastSignInAt = time.Now()
		tx.Save(user)

		a.issueRefreshToken(tx, user, w)
	} else {
		sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Password"})
	}
}

// RefreshTokenGrant implements the refresh_token grant type flow
func (a *API) RefreshTokenGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("refresh_token")

	if token == "" {
		sendJSON(w, 400, &OAuthError{Error: "invalid_request", Description: "refresh_token required"})
		return
	}

	tx := a.db.Begin()
	refreshToken := &models.RefreshToken{}
	if result := tx.First(refreshToken, "token = ?", token); result.Error != nil {
		tx.Rollback()
		if result.RecordNotFound() {
			sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Refresh Token"})
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	if refreshToken.Revoked {
		log.Printf("Possible abuse attempt: %v", r)
		sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Refresh Token"})
		tx.Rollback()
		return
	}

	// Make sure we revoke the current refreshtoken
	// (will be undone by a tx.Rollback if anything fails)
	refreshToken.Revoked = true
	tx.Save(refreshToken)

	user := &models.User{}
	if result := tx.Model(refreshToken).Related(user); result.Error != nil {
		tx.Rollback()
		if result.RecordNotFound() {
			sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Refresh Token"})
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	a.issueRefreshToken(tx, user, w)
}

func (a *API) generateAccessToken(user *models.User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	token.Claims["id"] = user.ID
	token.Claims["email"] = user.Email
	token.Claims["exp"] = time.Now().Add(time.Second * time.Duration(a.config.JWT.Exp)).Unix()
	for _, data := range user.Data {
		token.Claims[data.Key] = data.Value()
	}

	return token.SignedString([]byte(a.config.JWT.Secret))
}

func (a *API) issueRefreshToken(tx *gorm.DB, user *models.User, w http.ResponseWriter) {
	refreshToken, err := models.CreateRefreshToken(tx, user)
	if err != nil {
		tx.Rollback()
		InternalServerError(w, fmt.Sprintf("Error generating token: %v", err))
		return
	}

	user.Data = []models.Data{}
	tx.Model(user).Related(&user.Data)

	tokenString, err := a.generateAccessToken(user)

	if err != nil {
		tx.Rollback()
		InternalServerError(w, fmt.Sprintf("Error generating jwt token: %v", err))
		return
	}

	tx.Commit()

	sendJSON(w, 200, &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    a.config.JWT.Exp,
		RefreshToken: refreshToken.Token,
	})
}
