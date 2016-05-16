package api

import (
	"fmt"
	"net/http"

	"github.com/netlify/authlify/models"

	"golang.org/x/net/context"
)

// UserGet returns a user
func (a *API) UserGet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	token := getToken(ctx)

	user := &models.User{}
	if result := a.db.First(user, "id = ?", token.Claims["id"]); result.Error != nil {
		if result.RecordNotFound() {
			NotFoundError(w, "No user found for this token")
		} else {
			InternalServerError(w, fmt.Sprintf("Error during database query: %v", result.Error))
		}
		return
	}

	sendJSON(w, 200, user)
}

// UserUpdate updates fields on a user
func (a *API) UserUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) {

}
