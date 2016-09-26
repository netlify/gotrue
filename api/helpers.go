package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

// Error is an error with a message
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// OAuthError is the JSON handler for OAuth2 error responses
type OAuthError struct {
	Error       string `json:"error"`
	Description string `json:"description,omitempty"`
}

func sendJSON(w http.ResponseWriter, status int, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.Encode(obj)
}

func getToken(ctx context.Context) *jwt.Token {
	obj := ctx.Value("jwt")
	if obj == nil {
		return nil
	}
	return obj.(*jwt.Token)
}

// BadRequestError is simple Error Wrapper
func BadRequestError(w http.ResponseWriter, message string) {
	sendJSON(w, 400, &Error{Code: 400, Message: message})
}

// UnprocessableEntity is simple Error Wrapper
func UnprocessableEntity(w http.ResponseWriter, message string) {
	sendJSON(w, 422, &Error{Code: 422, Message: message})
}

// InternalServerError is simple Error Wrapper
func InternalServerError(w http.ResponseWriter, message string) {
	sendJSON(w, 500, &Error{Code: 500, Message: message})
}

// NotFoundError is simple Error Wrapper
func NotFoundError(w http.ResponseWriter, message string) {
	sendJSON(w, 404, &Error{Code: 404, Message: message})
}

// UnauthorizedError is simple Error Wrapper
func UnauthorizedError(w http.ResponseWriter, message string) {
	sendJSON(w, 401, &Error{Code: 401, Message: message})
}
