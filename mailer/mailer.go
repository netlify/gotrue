package mailer

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/mailme"

	"github.com/badoux/checkmail"
)

const defaultConfirmationMail = `<h2>Confirm your signup</h2>

<p>Follow this link to confirm your account:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your email address</a></p>`

const defaultRecoveryMail = `<h2>Reset password</h2>

<p>Follow this link to reset the password for your account:</p>
<p><a href="{{ .ConfirmationURL }}">Reset password</a></p>`

const defaultEmailChangeMail = `<h2>Confirm email address change</h2>

<p>Follow this link to confirm the update of your email address from {{ .Email }} to {{ .NewEmail }}:</p>
<p><a href="{{ .ConfirmationURL }}">Change email address</a></p>`

// Mailer defines the interface a mailer must implement.
type Mailer interface {
	Send(user *models.User, subject, body string, data map[string]interface{}) error
	ConfirmationMail(user *models.User) error
	RecoveryMail(user *models.User) error
	EmailChangeMail(user *models.User) error
	ValidateEmail(email string) error
}

// TemplateMailer will send mail and use templates from the site for easy mail styling
type TemplateMailer struct {
	SiteURL        string
	MemberFolder   string
	Config         *conf.Configuration
	TemplateMailer *mailme.Mailer
}

type noopMailer struct {
}

// MailSubjects holds the subject lines for the emails
type MailSubjects struct {
	ConfirmationMail string
	RecoveryMail     string
}

// NewMailer returns a new gotrue mailer
func NewMailer(conf *conf.Configuration) Mailer {
	if conf.Mailer.Host == "" {
		return &noopMailer{}
	}

	mailConf := conf.Mailer
	return &TemplateMailer{
		SiteURL:      mailConf.SiteURL,
		MemberFolder: mailConf.MemberFolder,
		Config:       conf,
		TemplateMailer: &mailme.Mailer{
			Host:    conf.Mailer.Host,
			Port:    conf.Mailer.Port,
			User:    conf.Mailer.User,
			Pass:    conf.Mailer.Pass,
			From:    conf.Mailer.AdminEmail,
			BaseURL: conf.Mailer.SiteURL,
		},
	}
}

// ValidateEmail returns nil if the email is valid,
// otherwise an error indicating the reason it is invalid
func (m TemplateMailer) ValidateEmail(email string) error {
	if err := checkmail.ValidateFormat(email); err != nil {
		return err
	}

	if err := checkmail.ValidateHost(email); err != nil {
		return err
	}

	return nil
}

// ConfirmationMail sends a signup confirmation mail to a new user
func (m *TemplateMailer) ConfirmationMail(user *models.User) error {
	return m.TemplateMailer.Mail(
		user.Email,
		withDefault(m.Config.Mailer.Subjects.Confirmation, "Confirm Your Signup"),
		m.Config.Mailer.Templates.Confirmation,
		defaultConfirmationMail,
		mailData("Confirmation", m.Config, user),
	)
}

// EmailChangeMail sends an email change confirmation mail to a user
func (m *TemplateMailer) EmailChangeMail(user *models.User) error {
	return m.TemplateMailer.Mail(
		user.EmailChange,
		withDefault(m.Config.Mailer.Subjects.EmailChange, "Confirm Email Change"),
		m.Config.Mailer.Templates.EmailChange,
		defaultEmailChangeMail,
		mailData("EmailChange", m.Config, user),
	)
}

// RecoveryMail sends a password recovery mail
func (m *TemplateMailer) RecoveryMail(user *models.User) error {
	return m.TemplateMailer.Mail(
		user.Email,
		withDefault(m.Config.Mailer.Subjects.Recovery, "Reset Your Password"),
		m.Config.Mailer.Templates.Recovery,
		defaultRecoveryMail,
		mailData("Recovery", m.Config, user),
	)
}

func mailData(mail string, config *conf.Configuration, user *models.User) map[string]interface{} {
	data := map[string]interface{}{
		"SiteURL":         config.Mailer.SiteURL,
		"ConfirmationURL": config.Mailer.SiteURL + config.Mailer.MemberFolder + "/confirm/" + user.ConfirmationToken,
		"Email":           user.Email,
		"Token":           user.ConfirmationToken,
		"Data":            user.UserMetaData,
	}

	// Setup recovery email
	if mail == "Recovery" {
		data["Token"] = user.RecoveryToken
		data["ConfirmationURL"] = config.Mailer.SiteURL + config.Mailer.MemberFolder + "/recover/" + user.RecoveryToken
	}

	// Setup email change confirmation email
	if mail == "EmailChange" {
		data["Token"] = user.EmailChangeToken
		data["ConfirmationURL"] = config.Mailer.SiteURL + config.Mailer.MemberFolder + "/confirm-email/" + user.EmailChangeToken
		data["NewEmail"] = user.EmailChange
	}
	return data
}

func withDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// Send can be used to send one-off emails to users
func (m TemplateMailer) Send(user *models.User, subject, body string, data map[string]interface{}) error {
	return m.TemplateMailer.Mail(
		user.Email,
		subject,
		"",
		body,
		data,
	)
}

func (m noopMailer) ValidateEmail(email string) error {
	return nil
}

func (m *noopMailer) ConfirmationMail(user *models.User) error {
	return nil
}

func (m noopMailer) RecoveryMail(user *models.User) error {
	return nil
}

func (m *noopMailer) EmailChangeMail(user *models.User) error {
	return nil
}

func (m noopMailer) Send(user *models.User, subject, body string, data map[string]interface{}) error {
	return nil
}
