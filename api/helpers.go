package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

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

func getUserFromClaims(ctx context.Context, conn storage.Connection) (*models.User, error) {
	claims := getClaims(ctx)
	if claims == nil {
		return nil, errors.New("Invalid token")
	}

	if claims.Subject == "" {
		return nil, errors.New("Invalid claim: id")
	}
	return conn.FindUserByID(claims.Subject)
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

func (a *API) isEmailBlacklisted(email string) bool {
	return a.blacklist.EmailBlacklisted(email)
}

func (a *API) isPasswordValid(password string) bool {
	re_min_nums := regexp.MustCompile("[0-9]")
	re_min_symbols := regexp.MustCompile("[$@$!%*#?&]")
	re_min_uppercase := regexp.MustCompile("[A-Z]")

	switch {
	case a.config.Password.MinNumbers > 0:
		if len(re_min_nums.FindAllString(password, -1)) < a.config.Password.MinNumbers {
			return false
		}
		fallthrough
	case a.config.Password.MinSymbols > 0:
		if len(re_min_symbols.FindAllString(password, -1)) < a.config.Password.MinSymbols {
			return false
		}
		fallthrough
	case a.config.Password.MinUppercase > 0:
		if len(re_min_uppercase.FindAllString(password, -1)) < a.config.Password.MinUppercase {
			return false
		}
		fallthrough
	default:
		if l := len(password); l == 0 || l < a.config.Password.MinLength {
			return false
		}
		return true
	}
}
