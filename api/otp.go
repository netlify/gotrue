package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/sethvargo/go-password/password"
)

// OtpParams contains the request body params for the otp endpoint
type OtpParams struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// SmsParams contains the request body params for sms otp
type SmsParams struct {
	Phone string `json:"phone"`
}

// Otp returns the MagicLink or SmsOtp handler based on the request body params
func (a *API) Otp(w http.ResponseWriter, r *http.Request) error {
	params := &OtpParams{}
	body, err := ioutil.ReadAll(r.Body)
	jsonDecoder := json.NewDecoder(bytes.NewReader(body))
	if err = jsonDecoder.Decode(params); err != nil {
		return badRequestError("Could not read verification params: %v", err)
	}
	if params.Email != "" && params.Phone != "" {
		return badRequestError("Only an email address or phone number should be provided")
	}

	r.Body = ioutil.NopCloser(strings.NewReader(string(body)))
	if params.Email != "" {
		return a.MagicLink(w, r)
	} else if params.Phone != "" {
		return a.SmsOtp(w, r)
	}

	return otpError("unsupported_otp_type", "")
}

// SmsOtp sends the user an otp via sms
func (a *API) SmsOtp(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)

	if !config.External.Phone.Enabled {
		return badRequestError("Unsupported phone provider")
	}

	instanceID := getInstanceID(ctx)
	params := &SmsParams{}
	jsonDecoder := json.NewDecoder(r.Body)
	if err := jsonDecoder.Decode(params); err != nil {
		return badRequestError("Could not read sms otp params: %v", err)
	}

	params.Phone = a.formatPhoneNumber(params.Phone)

	if isValid := a.validateE164Format(params.Phone); !isValid {
		return badRequestError("Invalid format: Phone number should follow the E.164 format")
	}

	aud := a.requestAud(ctx, r)

	user, uerr := models.FindUserByPhoneAndAudience(a.db, instanceID, params.Phone, aud)
	if uerr != nil {
		// if user does not exists, sign up the user
		if models.IsNotFoundError(uerr) {
			password, err := password.Generate(64, 10, 0, false, true)
			if err != nil {
				internalServerError("error creating user").WithInternalError(err)
			}
			newBodyContent := `{"phone":"` + params.Phone + `","password":"` + password + `"}`
			r.Body = ioutil.NopCloser(strings.NewReader(newBodyContent))
			r.ContentLength = int64(len(newBodyContent))

			fakeResponse := &responseStub{}

			if err := a.Signup(fakeResponse, r); err != nil {
				return err
			}
			return sendJSON(w, http.StatusOK, make(map[string]string))
		}
		return internalServerError("Database error finding user").WithInternalError(uerr)
	}

	err := a.db.Transaction(func(tx *storage.Connection) error {
		if err := models.NewAuditLogEntry(tx, instanceID, user, models.UserRecoveryRequestedAction, nil); err != nil {
			return err
		}

		if err := a.sendPhoneConfirmation(ctx, tx, user, params.Phone); err != nil {
			return badRequestError("Error sending sms otp: %v", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, make(map[string]string))
}
