package mailer

import "github.com/netlify/gotrue/models"

type noopMailer struct {
}

func (m noopMailer) ValidateEmail(email string) error {
	return nil
}

func (m *noopMailer) InviteMail(user *models.User, referrerURL string) error {
	return nil
}

func (m *noopMailer) ConfirmationMail(user *models.User, referrerURL string) error {
	return nil
}

func (m noopMailer) RecoveryMail(user *models.User, referrerURL string) error {
	return nil
}

func (m *noopMailer) EmailChangeMail(user *models.User, referrerURL string) error {
	return nil
}

func (m noopMailer) Send(user *models.User, subject, body string, data map[string]interface{}) error {
	return nil
}
