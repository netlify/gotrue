package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
)

var (
	MaxFrequencyLimitError error = errors.New("Frequency limit reached")
	configFile                   = ""
)

type GenerateLinkParams struct {
	Type       string                 `json:"type"`
	Email      string                 `json:"email"`
	Password   string                 `json:"password"`
	Data       map[string]interface{} `json:"data"`
	RedirectTo string                 `json:"redirect_to"`
}

func (a *API) GenerateLink(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := a.getConfig(ctx)
	mailer := a.Mailer(ctx)
	instanceID := getInstanceID(ctx)
	adminUser := getAdminUser(ctx)

	params := &GenerateLinkParams{}
	jsonDecoder := json.NewDecoder(r.Body)

	if err := jsonDecoder.Decode(params); err != nil {
		return badRequestError("Could not read body: %v", err)
	}

	if err := a.validateEmail(ctx, params.Email); err != nil {
		return err
	}

	aud := a.requestAud(ctx, r)
	user, err := models.FindUserByEmailAndAudience(a.db, instanceID, params.Email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			if params.Type == "magiclink" {
				params.Type = "signup"
				params.Password, err = password.Generate(64, 10, 0, false, true)
				if err != nil {
					return internalServerError("error creating user").WithInternalError(err)
				}
			} else if params.Type == "recovery" {
				return notFoundError(err.Error())
			}
		} else {
			return internalServerError("Database error finding user").WithInternalError(err)
		}
	}

	var url string
	referrer := a.getRedirectURLOrReferrer(r, params.RedirectTo)
	now := time.Now()
	err = a.db.Transaction(func(tx *storage.Connection) error {
		var terr error
		switch params.Type {
		case "magiclink", "recovery":
			if terr = models.NewAuditLogEntry(tx, instanceID, user, models.UserRecoveryRequestedAction, nil); terr != nil {
				return terr
			}
			user.RecoveryToken = crypto.SecureToken()
			user.RecoverySentAt = &now
			terr = errors.Wrap(tx.UpdateOnly(user, "recovery_token", "recovery_sent_at"), "Database error updating user for recovery")
		case "invite":
			if user != nil {
				if user.IsConfirmed() {
					return unprocessableEntityError(DuplicateEmailMsg)
				}
			} else {
				signupParams := &SignupParams{
					Email:    params.Email,
					Data:     params.Data,
					Provider: "email",
					Aud:      aud,
				}
				user, terr = a.signupNewUser(ctx, tx, signupParams)
				if terr != nil {
					return terr
				}
			}
			if terr = models.NewAuditLogEntry(tx, instanceID, adminUser, models.UserInvitedAction, map[string]interface{}{
				"user_id":    user.ID,
				"user_email": user.Email,
			}); terr != nil {
				return terr
			}
			user.ConfirmationToken = crypto.SecureToken()
			user.ConfirmationSentAt = &now
			user.InvitedAt = &now
			terr = errors.Wrap(tx.UpdateOnly(user, "confirmation_token", "confirmation_sent_at", "invited_at"), "Database error updating user for invite")
		case "signup":
			if user != nil {
				if user.IsConfirmed() {
					return unprocessableEntityError(DuplicateEmailMsg)
				}
				if err := user.UpdateUserMetaData(tx, params.Data); err != nil {
					return internalServerError("Database error updating user").WithInternalError(err)
				}
			} else {
				if params.Password == "" {
					return unprocessableEntityError("Signup requires a valid password")
				}
				if len(params.Password) < config.PasswordMinLength {
					return unprocessableEntityError(fmt.Sprintf("Password should be at least %d characters", config.PasswordMinLength))
				}
				signupParams := &SignupParams{
					Email:    params.Email,
					Password: params.Password,
					Data:     params.Data,
					Provider: "email",
					Aud:      aud,
				}
				user, terr = a.signupNewUser(ctx, tx, signupParams)
				if terr != nil {
					return terr
				}
			}
			user.ConfirmationToken = crypto.SecureToken()
			user.ConfirmationSentAt = &now
			terr = errors.Wrap(tx.UpdateOnly(user, "confirmation_token", "confirmation_sent_at"), "Database error updating user for confirmation")
		default:
			return badRequestError("Invalid email action link type requested: %v", params.Type)
		}

		if terr != nil {
			return terr
		}

		url, terr = mailer.GetEmailActionLink(user, params.Type, referrer)
		if terr != nil {
			return terr
		}
		return nil
	})

	if err != nil {
		return err
	}

	resp := make(map[string]interface{})
	u, err := json.Marshal(user)
	if err != nil {
		return internalServerError("User serialization error").WithInternalError(err)
	}
	if err = json.Unmarshal(u, &resp); err != nil {
		return internalServerError("User serialization error").WithInternalError(err)
	}
	resp["action_link"] = url

	return sendJSON(w, http.StatusOK, resp)
}

func sendConfirmation(tx *storage.Connection, u *models.User, mailer mailer.Mailer, maxFrequency time.Duration, referrerURL string) error {
	if u.ConfirmationSentAt != nil && !u.ConfirmationSentAt.Add(maxFrequency).Before(time.Now()) {
		return MaxFrequencyLimitError
	}

	oldToken := u.ConfirmationToken
	u.ConfirmationToken = crypto.SecureToken()
	now := time.Now()
	if err := mailer.ConfirmationMail(u, referrerURL); err != nil {
		u.ConfirmationToken = oldToken
		return errors.Wrap(err, "Error sending confirmation email")
	}
	u.ConfirmationSentAt = &now
	return errors.Wrap(tx.UpdateOnly(u, "confirmation_token", "confirmation_sent_at"), "Database error updating user for confirmation")
}

func sendInvite(tx *storage.Connection, u *models.User, mailer mailer.Mailer, referrerURL string) error {
	oldToken := u.ConfirmationToken
	u.ConfirmationToken = crypto.SecureToken()
	now := time.Now()
	if err := mailer.InviteMail(u, referrerURL); err != nil {
		u.ConfirmationToken = oldToken
		return errors.Wrap(err, "Error sending invite email")
	}
	u.InvitedAt = &now
	u.ConfirmationSentAt = &now
	return errors.Wrap(tx.UpdateOnly(u, "confirmation_token", "confirmation_sent_at", "invited_at"), "Database error updating user for invite")
}

func (a *API) sendPasswordRecovery(tx *storage.Connection, u *models.User, mailer mailer.Mailer, maxFrequency time.Duration, referrerURL string) error {
	if u.RecoverySentAt != nil && !u.RecoverySentAt.Add(maxFrequency).Before(time.Now()) {
		return MaxFrequencyLimitError
	}

	oldToken := u.RecoveryToken
	u.RecoveryToken = crypto.SecureToken()
	now := time.Now()
	if err := mailer.RecoveryMail(u, referrerURL); err != nil {
		u.RecoveryToken = oldToken
		return errors.Wrap(err, "Error sending recovery email")
	}
	u.RecoverySentAt = &now
	return errors.Wrap(tx.UpdateOnly(u, "recovery_token", "recovery_sent_at"), "Database error updating user for recovery")
}

func (a *API) sendMagicLink(tx *storage.Connection, u *models.User, mailer mailer.Mailer, maxFrequency time.Duration, referrerURL string) error {
	// since Magic Link is just a recovery with a different template and behaviour
	// around new users we will reuse the recovery db timer to prevent potential abuse
	if u.RecoverySentAt != nil && !u.RecoverySentAt.Add(maxFrequency).Before(time.Now()) {
		return MaxFrequencyLimitError
	}

	oldToken := u.RecoveryToken
	u.RecoveryToken = crypto.SecureToken()
	now := time.Now()
	if err := mailer.MagicLinkMail(u, referrerURL); err != nil {
		u.RecoveryToken = oldToken
		return errors.Wrap(err, "Error sending magic link email")
	}
	u.RecoverySentAt = &now
	return errors.Wrap(tx.UpdateOnly(u, "recovery_token", "recovery_sent_at"), "Database error updating user for recovery")
}

func (a *API) sendEmailChange(tx *storage.Connection, u *models.User, mailer mailer.Mailer, email string, referrerURL string) error {
	oldToken := u.EmailChangeToken
	oldEmail := u.EmailChange
	u.EmailChangeToken = crypto.SecureToken()
	u.EmailChange = email
	now := time.Now()
	if err := mailer.EmailChangeMail(u, referrerURL); err != nil {
		u.EmailChangeToken = oldToken
		u.EmailChange = oldEmail
		return err
	}

	u.EmailChangeSentAt = &now
	return errors.Wrap(tx.UpdateOnly(u, "email_change_token", "email_change", "email_change_sent_at"), "Database error updating user for email change")
}

func (a *API) validateEmail(ctx context.Context, email string) error {
	if email == "" {
		return unprocessableEntityError("An email address is required")
	}
	mailer := a.Mailer(ctx)
	if err := mailer.ValidateEmail(email); err != nil {
		return unprocessableEntityError("Unable to validate email address: " + err.Error())
	}
	return nil
}
