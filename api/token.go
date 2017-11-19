package api

import (
	"context"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
)

type GoTrueClaims struct {
	jwt.StandardClaims
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

const useCookieHeader = "x-use-cookie"
const useSessionCookie = "session"

// Token is the endpoint for OAuth access token requests
func (a *API) Token(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	grantType := r.FormValue("grant_type")

	switch grantType {
	case "password":
		return a.ResourceOwnerPasswordGrant(ctx, w, r)
	case "refresh_token":
		return a.RefreshTokenGrant(ctx, w, r)
	default:
		return oauthError("unsupported_grant_type", "")
	}
}

// ResourceOwnerPasswordGrant implements the password grant type flow
func (a *API) ResourceOwnerPasswordGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	username := r.FormValue("username")
	password := r.FormValue("password")
	cookie := r.Header.Get(useCookieHeader)

	aud := a.requestAud(ctx, r)
	instanceID := getInstanceID(ctx)
	config := a.getConfig(ctx)

	user, err := a.db.FindUserByEmailAndAudience(instanceID, username, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			return oauthError("invalid_grant", "No user found with this email")
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	if !user.IsConfirmed() {
		return oauthError("invalid_grant", "Email not confirmed")
	}

	if !user.Authenticate(password) {
		return oauthError("invalid_grant", "Invalid Password")
	}

	if cookie != "" && config.Cookie.Duration > 0 {
		if err = a.setCookieToken(config, user, cookie == useSessionCookie, w); err != nil {
			return internalServerError("Failed to set JWT cookie", err)
		}
	}

	if config.Webhook.HasEvent("login") {
		if err := triggerHook(LoginEvent, user, instanceID, config); err != nil {
			return err
		}
		a.db.UpdateUser(user)
	}

	return a.sendRefreshToken(ctx, user, w)
}

// RefreshTokenGrant implements the refresh_token grant type flow
func (a *API) RefreshTokenGrant(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	config := a.getConfig(ctx)
	tokenStr := r.FormValue("refresh_token")
	cookie := r.Header.Get(useCookieHeader)

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
		a.clearCookieToken(ctx, w)
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

	if cookie != "" && config.Cookie.Duration > 0 {
		if err = a.setCookieToken(config, user, cookie == useSessionCookie, w); err != nil {
			return internalServerError("Failed to set JWT cookie", err)
		}
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
			Subject:   user.ID,
			Audience:  user.Aud,
			ExpiresAt: time.Now().Add(expiresIn).Unix(),
		},
		Email:        user.Email,
		AppMetaData:  user.AppMetaData,
		UserMetaData: user.UserMetaData,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (a *API) issueRefreshToken(ctx context.Context, user *models.User) (*AccessTokenResponse, error) {
	config := a.getConfig(ctx)

	now := time.Now()
	user.LastSignInAt = &now

	refreshToken, err := a.db.GrantAuthenticatedUser(user)
	if err != nil {
		return nil, internalServerError("Database error granting user").WithInternalError(err)
	}

	tokenString, err := generateAccessToken(user, time.Second*time.Duration(config.JWT.Exp), config.JWT.Secret)
	if err != nil {
		a.db.RevokeToken(refreshToken)
		return nil, internalServerError("error generating jwt token").WithInternalError(err)
	}

	return &AccessTokenResponse{
		Token:        tokenString,
		TokenType:    "bearer",
		ExpiresIn:    config.JWT.Exp,
		RefreshToken: refreshToken.Token,
	}, nil
}

func (a *API) setCookieToken(config *conf.Configuration, user *models.User, session bool, w http.ResponseWriter) error {
	exp := time.Second * time.Duration(config.Cookie.Duration)

	tokenString, err := generateAccessToken(user, exp, config.JWT.Secret)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     config.Cookie.Key,
		Value:    tokenString,
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
	}
	if !session {
		cookie.Expires = time.Now().Add(exp)
		cookie.MaxAge = config.Cookie.Duration
	}

	http.SetCookie(w, cookie)
	return nil
}

func (a *API) clearCookieToken(ctx context.Context, w http.ResponseWriter) {
	config := getConfig(ctx)
	http.SetCookie(w, &http.Cookie{
		Name:     config.Cookie.Key,
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour * 10),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
	})
}

func (a *API) sendRefreshToken(ctx context.Context, user *models.User, w http.ResponseWriter) error {
	token, err := a.issueRefreshToken(ctx, user)
	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, token)
}
