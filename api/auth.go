package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
)

// requireAuthentication checks incoming requests for tokens presented using the Authorization header
func (api *API) requireAuthentication(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()
	config := getConfig(ctx)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, unauthorizedError("This endpoint requires a Bearer token")
	}

	matches := bearerRegexp.FindStringSubmatch(authHeader)
	if len(matches) != 2 {
		return nil, unauthorizedError("This endpoint requires a Bearer token")
	}

	token, err := jwt.Parse(matches[1], func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Method.Alg())
		}
		return []byte(config.JWT.Secret), nil
	})
	if err != nil {
		return nil, unauthorizedError("Invalid token: %v", err)
	}

	return withToken(ctx, token), nil
}

type adminCheckParams struct {
	User struct {
		Aud string `json:"aud"`
	} `json:"user"`
}

func (api *API) requireAdmin(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	// Find the administrative user
	ctx := r.Context()
	adminUser, err := getUser(ctx, api.db)
	if err != nil {
		return nil, unauthorizedError("Invalid admin user")
	}

	aud := api.requestAud(ctx, r)
	if r.Body != nil && r.Body != http.NoBody {
		params := adminCheckParams{}
		bod, err := r.GetBody()
		if err != nil {
			return nil, internalServerError("Error getting body").WithInternalError(err)
		}
		err = json.NewDecoder(bod).Decode(&params)
		if err != nil {
			return nil, badRequestError("Could not decode admin user params: %v", err)
		}
		if params.User.Aud != "" {
			aud = params.User.Aud
		}
	}

	// Make sure user is admin
	if !api.isAdmin(ctx, adminUser, aud) {
		return nil, unauthorizedError("User not allowed")
	}
	return ctx, nil
}
