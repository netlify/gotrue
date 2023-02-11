package api

import (
	"context"
	"time"

	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/models"
	"github.com/pkg/errors"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"github.com/tigrisdata/tigris-client-go/filter"
	"github.com/tigrisdata/tigris-client-go/fields"
)

func sendConfirmation(ctx context.Context, database *tigris.Database, u *models.User, mailer mailer.Mailer, maxFrequency time.Duration, referrerURL string) error {
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

	fieldsToSet, err := fields.UpdateBuilder().
		Set("confirmation_token", u.ConfirmationToken).
		Set("confirmation_sent_at", u.ConfirmationSentAt).
		Build()
	if err != nil {
		return err
	}

	if terr := u.BeforeUpdate(); terr != nil {
		return terr
	}

	_, err = tigris.GetCollection[models.User](database).Update(ctx, filter.Eq("id", u.ID), fieldsToSet)
	return errors.Wrap(err, "Database error updating user for confirmation")
}

func sendInvite(ctx context.Context, database *tigris.Database, u *models.User, mailer mailer.Mailer, referrerURL string) error {
	oldToken := u.ConfirmationToken
	u.ConfirmationToken = crypto.SecureToken()
	now := time.Now()
	if err := mailer.InviteMail(u, referrerURL); err != nil {
		u.ConfirmationToken = oldToken
		return errors.Wrap(err, "Error sending invite email")
	}
	u.InvitedAt = &now

	fieldsToSet, err := fields.UpdateBuilder().
		Set("confirmation_token", u.ConfirmationToken).
		Set("invited_at", u.InvitedAt).
		Build()
	if err != nil {
		return err
	}

	if terr := u.BeforeUpdate(); terr != nil {
		return terr
	}

	_, err = tigris.GetCollection[models.User](database).Update(ctx, filter.Eq("id", u.ID), fieldsToSet)
	return errors.Wrap(err, "Database error updating user for invite")
}

func (a *API) sendPasswordRecovery(ctx context.Context, database *tigris.Database, u *models.User, mailer mailer.Mailer, maxFrequency time.Duration, referrerURL string) error {
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

	fieldsToSet, err := fields.UpdateBuilder().
		Set("recovery_token", u.RecoveryToken).
		Set("recovery_sent_at", u.RecoverySentAt).
		Build()
	if err != nil {
		return err
	}

	if terr := u.BeforeUpdate(); terr != nil {
		return terr
	}

	_, err = tigris.GetCollection[models.User](database).Update(ctx, filter.Eq("id", u.ID), fieldsToSet)
	return errors.Wrap(err, "Database error updating user for recovery")
}

func (a *API) sendEmailChange(ctx context.Context, database *tigris.Database, u *models.User, mailer mailer.Mailer, email string, referrerURL string) error {
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

	fieldsToSet, err := fields.UpdateBuilder().
		Set("email_change_token", u.EmailChangeToken).
		Set("email_change", u.EmailChange).
		Set("email_change_sent_at", u.EmailChangeSentAt).
		Build()
	if err != nil {
		return err
	}

	if terr := u.BeforeUpdate(); terr != nil {
		return terr
	}

	_, err = tigris.GetCollection[models.User](database).Update(ctx, filter.Eq("id", u.ID), fieldsToSet)
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
