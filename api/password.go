package api

import "github.com/netlify/gotrue/models"

// validatePassword rejects passwords that violate GoTrue's server-side rules.
// Empty passwords are accepted here; callers that require a non-empty password
// (e.g. signup) check that separately.
func validatePassword(password string) error {
	if len(password) > models.MaxPasswordLength {
		return unprocessableEntityError("Password exceeds the maximum length of %d bytes", models.MaxPasswordLength)
	}
	return nil
}
