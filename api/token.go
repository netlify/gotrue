package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/netlify-auth/models"
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

	aud := a.requestAud(r)
	user, err := a.db.FindUserByEmailAndAudience(username, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "No user found with this email"})
		} else {
			InternalServerError(w, err.Error())
		}
		return
	}

	if !user.IsRegistered() {
		sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Email not confirmed"})
		return
	}

	if !user.Authenticate(password) {
		sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Password"})
		return
	}

	user.LastSignInAt = time.Now()
	a.issueRefreshToken(user, w)
}

// RefreshTokenGrant implements the refresh_token grant type flow
func (a *API) RefreshTokenGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tokenStr := r.FormValue("refresh_token")

	if tokenStr == "" {
		sendJSON(w, 400, &OAuthError{Error: "invalid_request", Description: "refresh_token required"})
		return
	}

	aud := a.requestAud(r)
	user, token, err := a.db.FindUserWithRefreshToken(tokenStr, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Refresh Token"})
		} else {
			InternalServerError(w, err.Error())
		}
	}

	if token.Revoked {
		log.Printf("Possible abuse attempt: %v", r)
		sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Refresh Token"})
		return
	}

	newToken, err := a.db.GrantRefreshTokenSwap(user, token)
	if err != nil {
		InternalServerError(w, err.Error())
	}

	tokenString, err := a.generateAccessToken(user)
	if err != nil {
		a.db.RollbackRefreshTokenSwap(newToken, token)
		InternalServerError(w, fmt.Sprintf("error generating jwt token: %v", err))
		return
	}

	sendJSON(w, 200, &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    a.config.JWT.Exp,
		RefreshToken: newToken.Token,
	})
}

func (a *API) generateAccessToken(user *models.User) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	token.Claims["id"] = user.ID
	token.Claims["email"] = user.Email
	token.Claims["aud"] = user.Aud
	token.Claims["exp"] = time.Now().Add(time.Second * time.Duration(a.config.JWT.Exp)).Unix()
	token.Claims["app_metadata"] = user.AppMetaData
	token.Claims["user_metadata"] = user.UserMetaData

	return token.SignedString([]byte(a.config.JWT.Secret))
}

func (a *API) issueRefreshToken(user *models.User, w http.ResponseWriter) {
	refreshToken, err := a.db.GrantAuthenticatedUser(user)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	tokenString, err := a.generateAccessToken(user)
	if err != nil {
		a.db.RevokeToken(refreshToken)
		InternalServerError(w, fmt.Sprintf("error generating jwt token: %v", err))
		return
	}

	sendJSON(w, 200, &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    a.config.JWT.Exp,
		RefreshToken: refreshToken.Token,
	})
}
