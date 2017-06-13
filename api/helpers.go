package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
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

func getUser(ctx context.Context, conn storage.Connection) (*models.User, error) {
	token := getToken(ctx)

	_id, ok := token.Claims["id"]
	if !ok {
		return nil, errors.New("Invalid claim: id")
	}

	id, ok := _id.(string)
	if !ok {
		return nil, errors.New("Invalid value for claim: id")
	}

	return conn.FindUserByID(id)
}

func (api *API) isAdmin(u *models.User, aud string) bool {
	if aud == "" {
		aud = api.config.JWT.Aud
	}
	return u.IsSuperAdmin || (aud == u.Aud && u.HasRole(api.config.JWT.AdminGroupName))
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
