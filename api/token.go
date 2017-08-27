package api

import (
	"context"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/models"
)

type GoTrueClaims struct {
	jwt.StandardClaims
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	AppMetaData  map[string]interface{} `json:"app_metadata"`
	UserMetaData map[string]interface{} `json:"user_metadata"`
}

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
	instanceID := getInstanceID(ctx)
	user, err := a.db.FindUserByEmailAndAudience(instanceID, username, aud)
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

	now := time.Now()
	user.LastSignInAt = &now
	return a.issueRefreshToken(ctx, user, w)
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

	provider, err := a.Provider(ctx, providerName)
	if err != nil {
		return badRequestError("Invalid provider: %s", providerName)
	}

	tok, err := provider.GetOAuthToken(ctx, code)
	if err != nil {
		return internalServerError("Unable to authenticate via %s", providerName).WithInternalError(err).WithInternalMessage("Error exchanging code with external provider")
	}

	data, err := provider.GetUserData(ctx, tok)
	if err != nil {
		return internalServerError("Error getting user email from %s", providerName).WithInternalError(err).WithInternalMessage("Error getting email address from external provider")
	}

	aud := a.requestAud(ctx, r)
	instanceID := getInstanceID(ctx)
	user, err := a.db.FindUserByEmailAndAudience(instanceID, data.Email, aud)
	if err != nil {
		return internalServerError(err.Error())
	}

	if !user.IsRegistered() {
		return oauthError("invalid_grant", "Email not confirmed")
	}

	now := time.Now()
	user.LastSignInAt = &now
	return a.issueRefreshToken(ctx, user, w)
}

// RefreshTokenGrant implements the refresh_token grant type flow
func (a *API) RefreshTokenGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	config := getConfig(ctx)
	tokenStr := r.FormValue("refresh_token")

	if tokenStr == "" {
		return oauthError("invalid_request", "refresh_token required")
	}

	user, token, err := a.db.FindUserWithRefreshToken(tokenStr)
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

	tokenString, err := generateAccessToken(user, time.Second*time.Duration(config.JWT.Exp), config.JWT.Secret)
	if err != nil {
		a.db.RollbackRefreshTokenSwap(newToken, token)
		return internalServerError("error generating jwt token").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    config.JWT.Exp,
		RefreshToken: newToken.Token,
	})
}

func generateAccessToken(user *models.User, expiresIn time.Duration, secret string) (string, error) {
	claims := &GoTrueClaims{
		StandardClaims: jwt.StandardClaims{
			Audience:  user.Aud,
			ExpiresAt: time.Now().Add(expiresIn).Unix(),
		},
		ID:           user.ID,
		Email:        user.Email,
		AppMetaData:  user.AppMetaData,
		UserMetaData: user.UserMetaData,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (a *API) issueRefreshToken(ctx context.Context, user *models.User, w http.ResponseWriter) error {
	config := getConfig(ctx)
	refreshToken, err := a.db.GrantAuthenticatedUser(user)
	if err != nil {
		return internalServerError("Database error granting user").WithInternalError(err)
	}

	tokenString, err := generateAccessToken(user, time.Second*time.Duration(config.JWT.Exp), config.JWT.Secret)
	if err != nil {
		a.db.RevokeToken(refreshToken)
		return internalServerError("error generating jwt token").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    config.JWT.Exp,
		RefreshToken: refreshToken.Token,
	})
}
