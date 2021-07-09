package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/sethvargo/go-password/password"
)

// MagicLinkParams holds the parameters for a magic link request
type MagicLinkParams struct {
	Email string `json:"email"`
}

// MagicLink sends a recovery email
func (a *API) MagicLink(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)

	if config.External.Email.Disabled {
		return badRequestError("Unsupported email provider")
	}

	instanceID := getInstanceID(ctx)
	params := &MagicLinkParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	err := jsonDecoder.Decode(params)
	if err != nil {
		return badRequestError("Could not read verification params: %v", err)
	}

	if params.Email == "" {
		return unprocessableEntityError("Password recovery requires an email")
	}
	if err := a.validateEmail(ctx, params.Email); err != nil {
		return err
	}

	aud := a.requestAud(ctx, r)
	user, err := models.FindUserByEmailAndAudience(a.db, instanceID, params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			// User doesn't exist, sign them up with temporary password
			password, err := password.Generate(64, 10, 0, false, true)
			if err != nil {
				internalServerError("error creating user").WithInternalError(err)
			}
			newBodyContent := `{"email":"` + params.Email + `","password":"` + password + `"}`
			r.Body = ioutil.NopCloser(strings.NewReader(newBodyContent))
			r.ContentLength = int64(len(newBodyContent))

			fakeResponse := &responseStub{}
			if config.Mailer.Autoconfirm {
				// signups are autoconfirmed, send magic link after signup
				if err := a.Signup(fakeResponse, r); err != nil {
					return err
				}
				newBodyContent := `{"email":"` + params.Email + `"}`
				r.Body = ioutil.NopCloser(strings.NewReader(newBodyContent))
				r.ContentLength = int64(len(newBodyContent))
				return a.MagicLink(w, r)
			}
			// otherwise confirmation email already contains 'magic link'
			if err := a.Signup(fakeResponse, r); err != nil {
				return err
			}

			return sendJSON(w, http.StatusOK, make(map[string]string))
		}
		return internalServerError("Database error finding user").WithInternalError(err)
	}

	err = a.db.Transaction(func(tx *storage.Connection) error {
		if terr := models.NewAuditLogEntry(tx, instanceID, user, models.UserRecoveryRequestedAction, nil); terr != nil {
			return terr
		}

		mailer := a.Mailer(ctx)
		referrer := a.getReferrer(r)
		return a.sendMagicLink(tx, user, mailer, config.SMTP.MaxFrequency, referrer)
	})
	if err != nil {
		if errors.Is(err, MaxFrequencyLimitError) {
			return tooManyRequestsError("For security purposes, you can only request this once every 60 seconds")
		}
		return internalServerError("Error sending magic link").WithInternalError(err)
	}

	return sendJSON(w, http.StatusOK, make(map[string]string))
}

// responseStub only implement http responsewriter for ignoring
// incoming data from methods where it passed
type responseStub struct {
}

func (rw *responseStub) Header() http.Header {
	return http.Header{}
}

func (rw *responseStub) Write(data []byte) (int, error) {
	return 1, nil
}

func (rw *responseStub) WriteHeader(statusCode int) {
}
