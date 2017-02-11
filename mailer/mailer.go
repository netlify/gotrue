package mailer

import (
	"github.com/netlify/mailme"
	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/models"
)

const DefaultConfirmationMail = `<h2>Confirm your signup</h2>

<p>Follow this link to confirm your account:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your mail</a></p>`
const DefaultRecoveryMail = `<h2>Reset Password</h2>

<p>Follow this link to reset the password for your account:</p>
<p><a href="{{ .ConfirmationURL }}">Reset Password</a></p>`

// Mailer will send mail and use templates from the site for easy mail styling
type Mailer struct {
	SiteURL        string
	MemberFolder   string
	Config         *conf.Configuration
	TemplateMailer *mailme.Mailer
}

// MailSubjects holds the subject lines for the emails
type MailSubjects struct {
	ConfirmationMail string
	RecoveryMail     string
}

// NewMailer returns a new netlify-auth mailer
func NewMailer(conf *conf.Configuration) *Mailer {
	mailConf := conf.Mailer
	return &Mailer{
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

// ConfirmationMail sends a signup confirmation mail to a new user
func (m *Mailer) ConfirmationMail(user *models.User) error {
	return m.TemplateMailer.Mail(
		user.Email,
		withDefault(m.Config.Mailer.Subjects.Confirmation, "Confirm Your Signup"),
		m.Config.Mailer.Templates.Confirmation,
		DefaultConfirmationMail,
		mailData("Confirmation", m.Config, user),
	)
}

// RecoveryMail sends a password recovery mail
func (m *Mailer) RecoveryMail(user *models.User) error {
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

	if mail == "Recovery" {
		data["Token"] = user.RecoveryToken
		data["ConfirmationURL"] = config.Mailer.SiteURL + config.Mailer.MemberFolder + "/recover/" + user.RecoveryToken
	}
	return data
}

func withDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
