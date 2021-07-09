package mailer

import (
	"github.com/badoux/checkmail"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/mailme"
)

// TemplateMailer will send mail and use templates from the site for easy mail styling
type TemplateMailer struct {
	SiteURL string
	Config  *conf.Configuration
	Mailer  *mailme.Mailer
}

var configFile = ""

const defaultInviteMail = `<h2>You have been invited</h2>

<p>You have been invited to create a user on {{ .SiteURL }}. Follow this link to accept the invite:</p>
<p><a href="{{ .ConfirmationURL }}">Accept the invite</a></p>`

const defaultConfirmationMail = `<h2>Confirm your signup</h2>

<p>Follow this link to confirm your user:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your email address</a></p>`

const defaultRecoveryMail = `<h2>Reset password</h2>

<p>Follow this link to reset the password for your user:</p>
<p><a href="{{ .ConfirmationURL }}">Reset password</a></p>`

const defaultMagicLinkMail = `<h2>Magic Link</h2>

<p>Follow this link to login:</p>
<p><a href="{{ .ConfirmationURL }}">Log In</a></p>`

const defaultEmailChangeMail = `<h2>Confirm email address change</h2>

<p>Follow this link to confirm the update of your email address from {{ .Email }} to {{ .NewEmail }}:</p>
<p><a href="{{ .ConfirmationURL }}">Change email address</a></p>`

// ValidateEmail returns nil if the email is valid,
// otherwise an error indicating the reason it is invalid
func (m TemplateMailer) ValidateEmail(email string) error {
	return checkmail.ValidateFormat(email)
}

// InviteMail sends a invite mail to a new user
func (m *TemplateMailer) InviteMail(user *models.User, referrerURL string) error {
	globalConfig, err := conf.LoadGlobal(configFile)

	redirectParam := ""
	if len(referrerURL) > 0 {
		redirectParam = "&redirect_to=" + referrerURL
	}

	url, err := getSiteURL(referrerURL, globalConfig.API.ExternalURL, m.Config.Mailer.URLPaths.Invite, "token="+user.ConfirmationToken+"&type=invite"+redirectParam)
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

	return m.Mailer.Mail(
		user.GetEmail(),
		string(withDefault(m.Config.Mailer.Subjects.Invite, "You have been invited")),
		m.Config.Mailer.Templates.Invite,
		defaultInviteMail,
		data,
	)
}

// ConfirmationMail sends a signup confirmation mail to a new user
func (m *TemplateMailer) ConfirmationMail(user *models.User, referrerURL string) error {
	globalConfig, err := conf.LoadGlobal(configFile)

	redirectParam := ""
	if len(referrerURL) > 0 {
		redirectParam = "&redirect_to=" + referrerURL
	}

	url, err := getSiteURL(referrerURL, globalConfig.API.ExternalURL, m.Config.Mailer.URLPaths.Confirmation, "token="+user.ConfirmationToken+"&type=signup"+redirectParam)
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

	return m.Mailer.Mail(
		user.GetEmail(),
		string(withDefault(m.Config.Mailer.Subjects.Confirmation, "Confirm Your Signup")),
		m.Config.Mailer.Templates.Confirmation,
		defaultConfirmationMail,
		data,
	)
}

// EmailChangeMail sends an email change confirmation mail to a user
func (m *TemplateMailer) EmailChangeMail(user *models.User, referrerURL string) error {
	redirectParam := ""
	if len(referrerURL) > 0 {
		redirectParam = "&redirect_to=" + referrerURL
	}

	url, err := getSiteURL(referrerURL, m.Config.SiteURL, m.Config.Mailer.URLPaths.EmailChange, "email_change_token="+user.EmailChangeToken+"&type=email_change"+redirectParam)
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"SiteURL":         m.Config.SiteURL,
		"ConfirmationURL": url,
		"Email":           user.GetEmail(),
		"NewEmail":        user.EmailChange,
		"Token":           user.EmailChangeToken,
		"Data":            user.UserMetaData,
	}

	return m.Mailer.Mail(
		user.EmailChange,
		string(withDefault(m.Config.Mailer.Subjects.EmailChange, "Confirm Email Change")),
		m.Config.Mailer.Templates.EmailChange,
		defaultEmailChangeMail,
		data,
	)
}

// RecoveryMail sends a password recovery mail
func (m *TemplateMailer) RecoveryMail(user *models.User, referrerURL string) error {
	globalConfig, err := conf.LoadGlobal(configFile)

	redirectParam := ""
	if len(referrerURL) > 0 {
		redirectParam = "&redirect_to=" + referrerURL
	}

	url, err := getSiteURL(referrerURL, globalConfig.API.ExternalURL, m.Config.Mailer.URLPaths.Recovery, "token="+user.RecoveryToken+"&type=recovery"+redirectParam)
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

	return m.Mailer.Mail(
		user.GetEmail(),
		string(withDefault(m.Config.Mailer.Subjects.Recovery, "Reset Your Password")),
		m.Config.Mailer.Templates.Recovery,
		defaultRecoveryMail,
		data,
	)
}

// MagicLinkMail sends a login link mail
func (m *TemplateMailer) MagicLinkMail(user *models.User, referrerURL string) error {
	globalConfig, err := conf.LoadGlobal(configFile)

	redirectParam := ""
	if len(referrerURL) > 0 {
		redirectParam = "&redirect_to=" + referrerURL
	}

	url, err := getSiteURL(referrerURL, globalConfig.API.ExternalURL, m.Config.Mailer.URLPaths.Recovery, "token="+user.RecoveryToken+"&type=magiclink"+redirectParam)
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

	return m.Mailer.Mail(
		user.GetEmail(),
		string(withDefault(m.Config.Mailer.Subjects.MagicLink, "Your Magic Link")),
		m.Config.Mailer.Templates.MagicLink,
		defaultMagicLinkMail,
		data,
	)
}

// Send can be used to send one-off emails to users
func (m TemplateMailer) Send(user *models.User, subject, body string, data map[string]interface{}) error {
	return m.Mailer.Mail(
		user.GetEmail(),
		subject,
		"",
		body,
		data,
	)
}
