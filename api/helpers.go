package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

func addRequestID(globalConfig *conf.GlobalConfiguration) middlewareHandler {
	return func(w http.ResponseWriter, r *http.Request) (context.Context, error) {
		id := ""
		if globalConfig.API.RequestIDHeader != "" {
			id = r.Header.Get(globalConfig.API.RequestIDHeader)
		}
		if id == "" {
			uid, err := uuid.NewV4()
			if err != nil {
				return nil, err
			}
			id = uid.String()
		}

		ctx := r.Context()
		ctx = withRequestID(ctx, id)
		return ctx, nil
	}
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

func getUserFromClaims(ctx context.Context, conn *storage.Connection) (*models.User, error) {
	claims := getClaims(ctx)
	if claims == nil {
		return nil, errors.New("Invalid token")
	}

	if claims.Subject == "" {
		return nil, errors.New("Invalid claim: id")
	}

	// System User
	instanceID := getInstanceID(ctx)

	if claims.Subject == models.SystemUserUUID.String() || claims.Subject == models.SystemUserID {
		return models.NewSystemUser(instanceID, claims.Audience), nil
	}
	userID, err := uuid.FromString(claims.Subject)
	if err != nil {
		return nil, errors.New("Invalid user ID")
	}
	return models.FindUserByInstanceIDAndID(conn, instanceID, userID)
}

func (a *API) isAdmin(ctx context.Context, u *models.User, aud string) bool {
	config := a.getConfig(ctx)
	if aud == "" {
		aud = config.JWT.Aud
	}
	return u.IsSuperAdmin || (aud == u.Aud && u.HasRole(config.JWT.AdminGroupName))
}

func (a *API) requestAud(ctx context.Context, r *http.Request) string {
	config := a.getConfig(ctx)
	// First check for an audience in the header
	if aud := r.Header.Get(audHeaderName); aud != "" {
		return aud
	}

	// Then check the token
	claims := getClaims(ctx)
	if claims != nil && claims.Audience != "" {
		return claims.Audience
	}

	// Finally, return the default of none of the above methods are successful
	return config.JWT.Aud
}

func (a *API) getReferrer(r *http.Request) string {
	ctx := r.Context()
	config := a.getConfig(ctx)
	referrer := ""
	if reqref := r.Referer(); reqref != "" {
		base, berr := url.Parse(config.SiteURL)
		refurl, rerr := url.Parse(reqref)
		// As long as the referrer came from the site, we will redirect back there
		if berr == nil && rerr == nil && base.Hostname() == refurl.Hostname() {
			referrer = reqref
		}
	}
	return referrer
}
