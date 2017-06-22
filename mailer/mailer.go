package mailer

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/mailme"

	"github.com/badoux/checkmail"
)

const DefaultConfirmationMail = `<h2>Confirm your signup</h2>

<p>Follow this link to confirm your account:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your mail</a></p>`

const DefaultRecoveryMail = `<h2>Reset Password</h2>

<p>Follow this link to reset the password for your account:</p>
<p><a href="{{ .ConfirmationURL }}">Reset Password</a></p>`

const DefaultEmailChangeMail = `<h2>Confirm Change of Email</h2>

<p>Follow this link to confirm the update of your email from {{ .Email }} to {{ .NewEmail }}:</p>
<p><a href="{{ .ConfirmationURL }}">Change Email</a></p>`

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

type NoOpMailer struct {
}

// MailSubjects holds the subject lines for the emails
type MailSubjects struct {
	ConfirmationMail string
	RecoveryMail     string
}

// NewMailer returns a new gotrue mailer
func NewMailer(conf *conf.Configuration) Mailer {
	if conf.Mailer.Host == "" {
		return &NoOpMailer{}
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
		DefaultConfirmationMail,
		mailData("Confirmation", m.Config, user),
	)
}

// EmailChangeMail sends an email change confirmation mail to a user
func (m *TemplateMailer) EmailChangeMail(user *models.User) error {
	return m.TemplateMailer.Mail(
		user.EmailChange,
		withDefault(m.Config.Mailer.Subjects.EmailChange, "Confirm Email Change"),
		m.Config.Mailer.Templates.EmailChange,
		DefaultEmailChangeMail,
		mailData("EmailChange", m.Config, user),
	)
}

// RecoveryMail sends a password recovery mail
func (m *TemplateMailer) RecoveryMail(user *models.User) error {
	return m.TemplateMailer.Mail(
		user.Email,
		withDefault(m.Config.Mailer.Subjects.Recovery, "Reset Your Password"),
		m.Config.Mailer.Templates.Recovery,
		DefaultRecoveryMail,
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

func (m NoOpMailer) ValidateEmail(email string) error {
	return nil
}

func (m *NoOpMailer) ConfirmationMail(user *models.User) error {
	return nil
}

func (m NoOpMailer) RecoveryMail(user *models.User) error {
	return nil
}

func (m *NoOpMailer) EmailChangeMail(user *models.User) error {
	return nil
}

// Send does nothing for NoOpMailer
func (m NoOpMailer) Send(user *models.User, subject, body string, data map[string]interface{}) error {
	return nil
}
