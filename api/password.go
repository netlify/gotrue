package api

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
)

// validatePassword applies the server-side password rules:
//   - rejects anything longer than bcrypt's input limit (always-on)
//   - enforces a minimum length when the instance opts in to strict security
//
// Empty passwords are accepted here; callers that require a non-empty
// password (e.g. signup) check that separately.
func validatePassword(config *conf.Configuration, password string) error {
	if password == "" {
		return nil
	}
	if len(password) > models.MaxPasswordLength {
		return unprocessableEntityError("Password exceeds the maximum length of %d bytes", models.MaxPasswordLength)
	}
	if config.Security.Strict && len(password) < config.Security.MinPasswordLength {
		return unprocessableEntityError(
			"Password must be at least %d characters long",
			config.Security.MinPasswordLength,
		)
	}
	return nil
}
