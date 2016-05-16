package api

import (
	"encoding/json"
	"net/http"
)

// Error is an error with a message
type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func sendJSON(w http.ResponseWriter, status int, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.Encode(obj)
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
