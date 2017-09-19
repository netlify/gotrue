package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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

var minNumsRegexp = regexp.MustCompile("[[:digit:]]")
var minSymbolsRegexp = regexp.MustCompile("[$@$!%*#?&]")
var minUppercaseRegexp = regexp.MustCompile("[[:upper:]]")
var minLowercaseRegexp = regexp.MustCompile("[[:lower:]]")

func (a *API) isPasswordValid(password string) (string, bool) {
	errors := []string{}
	config := a.config.Password

	if l := len(password); l == 0 || l < config.MinLength {
		errors = append(errors, fmt.Sprintf("at least %d chars", config.MinLength))
	}

	if len(minNumsRegexp.FindAllString(password, -1)) < config.MinNumbers {
		errors = append(errors, fmt.Sprintf("contain at least %d numbers", config.MinNumbers))
	}

	if len(minSymbolsRegexp.FindAllString(password, -1)) < config.MinSymbols {
		errors = append(errors, fmt.Sprintf("contain at least %d symbols", config.MinSymbols))
	}

	if len(minUppercaseRegexp.FindAllString(password, -1)) < config.MinUppercase {
		errors = append(errors, fmt.Sprintf("contain at least %d uppercase characters", config.MinUppercase))
	}

	if len(minLowercaseRegexp.FindAllString(password, -1)) < config.MinLowercase {
		errors = append(errors, fmt.Sprintf("contain at least %d lowercase characters", config.MinLowercase))
	}

	if len(errors) == 0 {
		return "", true
	}

	return fmt.Sprintf("Password must %s", strings.Join(errors, ", ")), false
}
