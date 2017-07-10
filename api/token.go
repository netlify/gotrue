package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/models"
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
	case "authorization_code":
		a.AuthorizationCodeGrant(ctx, w, r)
	default:
		sendJSON(w, 400, &OAuthError{Error: "unsupported_grant_type"})
	}
}

// ResourceOwnerPasswordGrant implements the password grant type flow
func (a *API) ResourceOwnerPasswordGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	aud := a.requestAud(ctx, r)
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

// AuthorizationCodeGrant implements the authorization_code grant for use with external OAuth providers
func (a *API) AuthorizationCodeGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	providerName := r.FormValue("provider")

	if code == "" {
		logrus.Warnf("No authorization code found: %v", r)
		sendJSON(w, 400, &OAuthError{Error: "invalid_request", Description: "Authorization code required"})
		return
	} else if providerName == "" {
		logrus.Warnf("No provider name found: %v", r)
		sendJSON(w, 400, &OAuthError{Error: "invalid_request", Description: "External provider name required"})
		return
	}

	provider, err := a.Provider(providerName)
	if err != nil {
		logrus.Warnf("Error finding provider: %+v", err)
		BadRequestError(w, fmt.Sprintf("Invalid provider: %s", providerName))
		return
	}

	tok, err := provider.GetOAuthToken(ctx, code)
	if err != nil {
		logrus.Warnf("Error exchanging code with external provider %+v", err.Error())
		InternalServerError(w, fmt.Sprintf("Unable to authenticate via %s", providerName))
		return
	}

	email, err := provider.GetUserEmail(ctx, tok)
	if err != nil {
		logrus.Warnf("Error getting email address from external provider: %+v", err)
		InternalServerError(w, fmt.Sprintf("Error getting user email from %s", providerName))
		return
	}

	aud := a.requestAud(ctx, r)
	user, err := a.db.FindUserByEmailAndAudience(email, aud)
	if err != nil {
		InternalServerError(w, err.Error())
		return
	}

	if !user.IsRegistered() {
		sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Email not confirmed"})
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

	aud := a.requestAud(ctx, r)
	user, token, err := a.db.FindUserWithRefreshToken(tokenStr, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			sendJSON(w, 400, &OAuthError{Error: "invalid_grant", Description: "Invalid Refresh Token"})
		} else {
			InternalServerError(w, err.Error())
		}
	}

	if token.Revoked {
		logrus.Warnf("Possible abuse attempt: %v", r)
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
