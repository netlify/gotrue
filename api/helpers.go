package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

func addRequestID(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	id := uuid.NewRandom().String()
	ctx := r.Context()
	ctx = withRequestID(ctx, id)
	return ctx, nil
}

func sendJSON(w http.ResponseWriter, status int, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error encoding json response: %v", obj))
	}
	w.WriteHeader(status)
	_, err = w.Write(b)
	return err
}

func getUser(ctx context.Context, conn storage.Connection) (*models.User, error) {
	token := getToken(ctx)
	if token == nil {
		return nil, errors.New("Invalid token")
	}

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

func (api *API) requestAud(ctx context.Context, r *http.Request) string {
	// First check for an audience in the header
	if aud := r.Header.Get(audHeaderName); aud != "" {
		return aud
	}

	// Then check the token
	token := getToken(ctx)
	if token != nil {
		if _aud, ok := token.Claims["aud"]; ok {
			if aud, ok := _aud.(string); ok && aud != "" {
				return aud
			}
		}
	}

	// Finally, return the default of none of the above methods are successful
	return api.config.JWT.Aud
}
