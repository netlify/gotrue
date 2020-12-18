package api

import (
	"context"
	"time"

	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
)

func sendConfirmation(tx *storage.Connection, u *models.User, mailer mailer.Mailer, maxFrequency time.Duration, referrerURL string) error {
	if u.ConfirmationSentAt != nil && !u.ConfirmationSentAt.Add(maxFrequency).Before(time.Now()) {
		return nil
	}

	oldToken := u.ConfirmationToken
	u.ConfirmationToken = crypto.SecureToken()
	now := time.Now()
	if err := mailer.ConfirmationMail(u, referrerURL); err != nil {
		u.ConfirmationToken = oldToken
		return errors.Wrap(err, "Error sending confirmation email")
	}
	u.ConfirmationSentAt = &now
	err := tx.Model(&u).Select("confirmation_token", "confirmation_sent_at").Updates(u).Error
	return errors.Wrap(err, "Database error updating user for confirmation")
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
	err := tx.Model(&u).Select("confirmation_token", "invited_at").Updates(u).Error
	return errors.Wrap(err, "Database error updating user for invite")
}

func (a *API) sendPasswordRecovery(tx *storage.Connection, u *models.User, mailer mailer.Mailer, maxFrequency time.Duration, referrerURL string) error {
	if u.RecoverySentAt != nil && !u.RecoverySentAt.Add(maxFrequency).Before(time.Now()) {
		return nil
	}

	oldToken := u.RecoveryToken
	u.RecoveryToken = crypto.SecureToken()
	now := time.Now()
	if err := mailer.RecoveryMail(u, referrerURL); err != nil {
		u.RecoveryToken = oldToken
		return errors.Wrap(err, "Error sending recovery email")
	}
	u.RecoverySentAt = &now
	err := tx.Model(&u).Select("recovery_token", "recovery_sent_at").Updates(u).Error
	return errors.Wrap(err, "Database error updating user for recovery")
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
	err := tx.Model(&u).Select("email_change_token", "email_change", "email_change_sent_at").Updates(u).Error
	return errors.Wrap(err, "Database error updating user for email change")
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
