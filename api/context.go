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
	tokenKey     = contextKey("jwt")
	requestIDKey = contextKey("request_id")
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

// withRequestID adds the provided request ID to the context.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// getRequestID reads the request ID from the context.
func getRequestID(ctx context.Context) string {
	obj := ctx.Value(requestIDKey)
	if obj == nil {
		return ""
	}

	return obj.(string)
}
