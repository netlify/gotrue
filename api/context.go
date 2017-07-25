package api

import (
	"context"

	jwt "github.com/dgrijalva/jwt-go"
)

type contextKey string

func (c contextKey) String() string {
	return "api context key " + string(c)
}

const (
	tokenKey = contextKey("jwt")
)

// WithToken adds the JWT token to the context.
func withToken(ctx context.Context, token *jwt.Token) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// GetToken reads the JWT token from the context.
func getToken(ctx context.Context) *jwt.Token {
	obj := ctx.Value(tokenKey)
	if obj == nil {
		return nil
	}

	return obj.(*jwt.Token)
}
