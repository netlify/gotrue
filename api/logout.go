package api

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

var bearerRegexp = regexp.MustCompile(`^(?:B|b)earer (\S+$)`)

// Logout is the endpoint for logging out a user and thereby revoking any refresh tokens
func (a *API) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		UnauthorizedError(w, "This endpoint requires a Bearer token")
		return
	}

	matches := bearerRegexp.FindStringSubmatch(authHeader)
	if len(matches) != 2 {
		UnauthorizedError(w, "This endpoint requires a Bearer token")
		return
	}

	token, err := jwt.Parse(matches[1], func(token *jwt.Token) (interface{}, error) {
		if token.Header["alg"] != "HS256" {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.config.JWT.Secret), nil
	})
	if err != nil {
		UnauthorizedError(w, fmt.Sprintf("Invalid token: %v", err))
		return
	}

	a.db.Where("user_id = ?", token.Claims["id"]).Delete(&models.RefreshToken{})

	w.WriteHeader(204)
}
