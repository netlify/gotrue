package mailer

import (
	"net/url"
	"time"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/mailme"

	"github.com/badoux/checkmail"
)

const defaultInviteMail = `<h2>You have been invited</h2>

<p>You have been invited to create a user on {{ .SiteURL }}. Follow this link to accept the invite:</p>
<p><a href="{{ .ConfirmationURL }}">Accept the invite</a></p>`

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
	InviteMail(user *models.User) error
	ConfirmationMail(user *models.User) error
	RecoveryMail(user *models.User) error
	EmailChangeMail(user *models.User) error
	ValidateEmail(email string) error
}

// TemplateMailer will send mail and use templates from the site for easy mail styling
type TemplateMailer struct {
	SiteURL        string
	Config         *conf.Configuration
	TemplateMailer *mailme.Mailer
	MaxFrequency   time.Duration
}

type noopMailer struct {
}

// MailSubjects holds the subject lines for the emails
type MailSubjects struct {
	ConfirmationMail string
	RecoveryMail     string
}

// NewMailer returns a new gotrue mailer
func NewMailer(smtp conf.SMTPConfiguration, instanceConfig *conf.Configuration) Mailer {
	if smtp.Host == "" && instanceConfig.SMTP.Host == "" {
		return &noopMailer{}
	}

	smtpHost := instanceConfig.SMTP.Host
	if smtpHost == "" {
		smtpHost = smtp.Host
	}
	smtpPort := instanceConfig.SMTP.Port
	if smtpPort == 0 {
		smtpPort = smtp.Port
	}
	smtpUser := instanceConfig.SMTP.User
	if smtpUser == "" {
		smtpUser = smtp.User
	}
	smtpPass := instanceConfig.SMTP.Pass
	if smtpPass == "" {
		smtpPass = smtp.Pass
	}
	smtpAdminEmail := instanceConfig.SMTP.AdminEmail
	if smtpAdminEmail == "" {
		smtpAdminEmail = smtp.AdminEmail
	}
	smtpMaxFrequency := instanceConfig.SMTP.MaxFrequency
	if smtpMaxFrequency == 0 {
		smtpMaxFrequency = smtp.MaxFrequency
	}

	return &TemplateMailer{
		SiteURL:      instanceConfig.SiteURL,
		MaxFrequency: smtpMaxFrequency,
		Config:       instanceConfig,
		TemplateMailer: &mailme.Mailer{
			Host:    smtpHost,
			Port:    smtpPort,
			User:    smtpUser,
			Pass:    smtpPass,
			From:    smtpAdminEmail,
			BaseURL: instanceConfig.SiteURL,
		},
	}
}

// ValidateEmail returns nil if the email is valid,
// otherwise an error indicating the reason it is invalid
func (m TemplateMailer) ValidateEmail(email string) error {
	return checkmail.ValidateFormat(email)
}

// InviteMail sends a invite mail to a new user
func (m *TemplateMailer) InviteMail(user *models.User) error {
	if user.ConfirmationSentAt != nil && !user.ConfirmationSentAt.Add(m.MaxFrequency).Before(time.Now()) {
		return nil
	}

	url, err := getSiteURL(m.Config.SiteURL, m.Config.Mailer.URLPaths.Invite, "invite_token="+user.ConfirmationToken)
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": url,
		"Email":           user.Email,
		"Token":           user.ConfirmationToken,
		"Data":            user.UserMetaData,
	}

	return m.TemplateMailer.Mail(
		user.Email,
		withDefault(m.Config.Mailer.Subjects.Invite, "You have been invited"),
		m.Config.Mailer.Templates.Invite,
		defaultInviteMail,
		data,
	)
}

// ConfirmationMail sends a signup confirmation mail to a new user
func (m *TemplateMailer) ConfirmationMail(user *models.User) error {
	if user.ConfirmationSentAt != nil && !user.ConfirmationSentAt.Add(m.MaxFrequency).Before(time.Now()) {
		return nil
	}

	url, err := getSiteURL(m.Config.SiteURL, m.Config.Mailer.URLPaths.Confirmation, "confirmation_token="+user.ConfirmationToken)
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": url,
		"Email":           user.Email,
		"Token":           user.ConfirmationToken,
		"Data":            user.UserMetaData,
	}

	return m.TemplateMailer.Mail(
		user.Email,
		withDefault(m.Config.Mailer.Subjects.Confirmation, "Confirm Your Signup"),
		m.Config.Mailer.Templates.Confirmation,
		defaultConfirmationMail,
		data,
	)
}

// EmailChangeMail sends an email change confirmation mail to a user
func (m *TemplateMailer) EmailChangeMail(user *models.User) error {
	url, err := getSiteURL(m.Config.SiteURL, m.Config.Mailer.URLPaths.EmailChange, "email_change_token="+user.EmailChangeToken)
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": url,
		"Email":           user.Email,
		"NewEmail":        user.EmailChange,
		"Token":           user.EmailChangeToken,
		"Data":            user.UserMetaData,
	}

	return m.TemplateMailer.Mail(
		user.EmailChange,
		withDefault(m.Config.Mailer.Subjects.EmailChange, "Confirm Email Change"),
		m.Config.Mailer.Templates.EmailChange,
		defaultEmailChangeMail,
		data,
	)
}

// RecoveryMail sends a password recovery mail
func (m *TemplateMailer) RecoveryMail(user *models.User) error {
	url, err := getSiteURL(m.Config.SiteURL, m.Config.Mailer.URLPaths.Recovery, "recovery_token="+user.RecoveryToken)
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": url,
		"Email":           user.Email,
		"Token":           user.RecoveryToken,
		"Data":            user.UserMetaData,
	}

	return m.TemplateMailer.Mail(
		user.Email,
		withDefault(m.Config.Mailer.Subjects.Recovery, "Reset Your Password"),
		m.Config.Mailer.Templates.Recovery,
		defaultRecoveryMail,
		data,
	)
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

func (m *noopMailer) InviteMail(user *models.User) error {
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

func getSiteURL(siteURL, filepath, fragment string) (string, error) {
	site, err := url.Parse(siteURL)
	if err != nil {
		return "", err
	}
	path, err := url.Parse(filepath)
	if err != nil {
		return "", err
	}
	u := site.ResolveReference(path)
	u.Fragment = fragment
	return u.String(), nil
}
