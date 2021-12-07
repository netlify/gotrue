package api

import (
	"context"
	"encoding/json"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v4"
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

type adminCheckParams struct {
	Aud string `json:"aud"`
}

func (a *API) requireAdmin(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
	// Find the administrative user
	adminUser, err := getUserFromClaims(ctx, a.db)
	if err != nil {
		return nil, unauthorizedError("Invalid admin user").WithInternalError(err)
	}

	aud := a.requestAud(ctx, r)
	if r.Body != nil && r.Body != http.NoBody {
		c, err := addGetBody(w, r)
		if err != nil {
			return nil, internalServerError("Error getting body").WithInternalError(err)
		}
		ctx = c

		params := adminCheckParams{}
		bod, err := r.GetBody()
		if err != nil {
			return nil, internalServerError("Error getting body").WithInternalError(err)
		}
		err = json.NewDecoder(bod).Decode(&params)
		if err != nil {
			return nil, badRequestError("Could not decode admin user params: %v", err)
		}
		if params.Aud != "" {
			aud = params.Aud
		}
	}

	// Make sure user is admin
	if !a.isAdmin(ctx, adminUser, aud) {
		return nil, unauthorizedError("User not allowed")
	}
	return withAdminUser(ctx, adminUser), nil
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
