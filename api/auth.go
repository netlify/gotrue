package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/netlify/gotrue/models"
)

// requireAuthentication checks incoming requests for tokens presented using the Authorization header
func (a *API) requireAuthentication(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	token, err := a.extractBearerToken(w, r)
	if err != nil {
		a.clearCookieToken(r.Context(), w)
		return nil, err
	}

	return a.parseJWTClaims(token, r, w)
}

func (a *API) requireAdmin(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
	// Find the administrative user
	claims := getClaims(ctx)
	if claims == nil {
		fmt.Printf("[%s] %s %s %d %s\n", time.Now().Format("2006-01-02 15:04:05"), r.Method, r.RequestURI, http.StatusForbidden, "Invalid token")
		return nil, unauthorizedError("Invalid token")
	}

	adminRoles := a.getConfig(ctx).JWT.AdminRoles

	if isStringInSlice(claims.Role, adminRoles) {
		// successful authentication
		return withAdminUser(ctx, &models.User{}), nil
	}

	fmt.Printf("[%s] %s %s %d %s\n", time.Now().Format("2006-01-02 15:04:05"), r.Method, r.RequestURI, http.StatusForbidden, "this token needs role 'supabase_admin' or 'service_role'")
	return nil, unauthorizedError("User not allowed")
}

func (a *API) extractBearerToken(w http.ResponseWriter, r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", unauthorizedError("This endpoint requires a Bearer token")
	}

	matches := bearerRegexp.FindStringSubmatch(authHeader)
	if len(matches) != 2 {
		return "", unauthorizedError("This endpoint requires a Bearer token")
	}

	return matches[1], nil
}

func (a *API) parseJWTClaims(bearer string, r *http.Request, w http.ResponseWriter) (context.Context, error) {
	ctx := r.Context()
	config := a.getConfig(ctx)

	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	token, err := p.ParseWithClaims(bearer, &GoTrueClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT.Secret), nil
	})
	if err != nil {
		a.clearCookieToken(ctx, w)
		return nil, unauthorizedError("Invalid token: %v", err)
	}

	return withToken(ctx, token), nil
}
