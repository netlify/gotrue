package api

import (
	"context"
	"net/http"
	"time"

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
func (a *API) Token(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	grantType := r.FormValue("grant_type")

	switch grantType {
	case "password":
		return a.ResourceOwnerPasswordGrant(ctx, w, r)
	case "refresh_token":
		return a.RefreshTokenGrant(ctx, w, r)
	case "authorization_code":
		return a.AuthorizationCodeGrant(ctx, w, r)
	default:
		return oauthError("unsupported_grant_type", "")
	}
}

// ResourceOwnerPasswordGrant implements the password grant type flow
func (a *API) ResourceOwnerPasswordGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	username := r.FormValue("username")
	password := r.FormValue("password")

	aud := a.requestAud(ctx, r)
	user, err := a.db.FindUserByEmailAndAudience(username, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			return oauthError("invalid_grant", "No user found with this email")
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	if !user.IsRegistered() {
		return oauthError("invalid_grant", "Email not confirmed")
	}

	if !user.Authenticate(password) {
		return oauthError("invalid_grant", "Invalid Password")
	}

	user.LastSignInAt = time.Now()
	return a.issueRefreshToken(user, w)
}

// AuthorizationCodeGrant implements the authorization_code grant for use with external OAuth providers
func (a *API) AuthorizationCodeGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	code := r.FormValue("code")
	providerName := r.FormValue("provider")

	if code == "" {
		return oauthError("invalid_request", "Authorization code required").WithInternalMessage("No authorization code found: %v", r)
	} else if providerName == "" {
		return oauthError("invalid_request", "External provider name required").WithInternalMessage("No provider name found: %v", r)
	}

	provider, err := a.Provider(providerName)
	if err != nil {
		return badRequestError("Invalid provider: %s", providerName)
	}

	tok, err := provider.GetOAuthToken(ctx, code)
	if err != nil {
		return internalServerError("Unable to authenticate via %s", providerName).WithInternalError(err).WithInternalMessage("Error exchanging code with external provider")
	}

	email, err := provider.GetUserEmail(ctx, tok)
	if err != nil {
		return internalServerError("Error getting user email from %s", providerName).WithInternalError(err).WithInternalMessage("Error getting email address from external provider")
	}

	aud := a.requestAud(ctx, r)
	user, err := a.db.FindUserByEmailAndAudience(email, aud)
	if err != nil {
		return internalServerError(err.Error())
	}

	if !user.IsRegistered() {
		return oauthError("invalid_grant", "Email not confirmed")
	}

	user.LastSignInAt = time.Now()
	return a.issueRefreshToken(user, w)
}

// RefreshTokenGrant implements the refresh_token grant type flow
func (a *API) RefreshTokenGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	tokenStr := r.FormValue("refresh_token")

	if tokenStr == "" {
		return oauthError("invalid_request", "refresh_token required")
	}

	aud := a.requestAud(ctx, r)
	user, token, err := a.db.FindUserWithRefreshToken(tokenStr, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			return oauthError("invalid_grant", "Invalid Refresh Token")
		}
		return internalServerError(err.Error())
	}

	if token.Revoked {
		return oauthError("invalid_grant", "Invalid Refresh Token").WithInternalMessage("Possible abuse attempt: %v", r)
	}

	newToken, err := a.db.GrantRefreshTokenSwap(user, token)
	if err != nil {
		return internalServerError(err.Error())
	}

	tokenString, err := a.generateAccessToken(user)
	if err != nil {
		a.db.RollbackRefreshTokenSwap(newToken, token)
		return internalServerError("error generating jwt token").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, &AccessTokenResponse{
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

func (a *API) issueRefreshToken(user *models.User, w http.ResponseWriter) error {
	refreshToken, err := a.db.GrantAuthenticatedUser(user)
	if err != nil {
		return internalServerError("Database error granting user").WithInternalError(err)
	}

	tokenString, err := a.generateAccessToken(user)
	if err != nil {
		a.db.RevokeToken(refreshToken)
		return internalServerError("error generating jwt token").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    a.config.JWT.Exp,
		RefreshToken: refreshToken.Token,
	})
}
