package mailer

import (
	"gopkg.in/gomail.v2"

	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/models"
)

// Mailer will send mail and use templates from the site for easy mail styling
type Mailer struct {
	SiteURL        string
	TemplateFolder string
	MemberFolder   string
	Host           string
	Port           int
	User           string
	Pass           string
	AdminEmail     string
	MailSubjects   MailSubjects
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
		SiteURL:        mailConf.SiteURL,
		TemplateFolder: mailConf.TemplateFolder,
		MemberFolder:   mailConf.MemberFolder,
		Host:           mailConf.Host,
		Port:           mailConf.Port,
		User:           mailConf.User,
		Pass:           mailConf.Pass,
		AdminEmail:     mailConf.AdminEmail,
		MailSubjects: MailSubjects{
			ConfirmationMail: mailConf.MailSubjects.ConfirmationMail,
			RecoveryMail:     mailConf.MailSubjects.RecoveryMail,
		},
	}
}

// ConfirmationMail sends a signup confirmation mail to a new user
func (m *Mailer) ConfirmationMail(user *models.User) error {
	confirmationURL := m.SiteURL + m.MemberFolder + "/confirm/" + user.ConfirmationToken

	mail := gomail.NewMessage()
	mail.SetHeader("From", m.AdminEmail)
	mail.SetHeader("To", user.Email)
	mail.SetHeader("Subject", m.MailSubjects.ConfirmationMail)
	mail.SetBody("text/html", `<h2>Please verify your registration</h2>

<p>Follow this link to complete the registration process:</p>
<p><a href="`+confirmationURL+`">Complete Registration</a></p>`)

	dial := gomail.NewPlainDialer(m.Host, m.Port, m.User, m.Pass)
	return dial.DialAndSend(mail)
}

// RecoveryMail sends a password recovery mail
func (m *Mailer) RecoveryMail(user *models.User) error {
	confirmationURL := m.SiteURL + m.MemberFolder + "/recover/" + user.RecoveryToken

	mail := gomail.NewMessage()
	mail.SetHeader("From", m.AdminEmail)
	mail.SetHeader("To", user.Email)
	mail.SetHeader("Subject", m.MailSubjects.RecoveryMail)
	mail.SetBody("text/html", `<h2>Recover your password</h2>

<p>Follow this link to reset your password:</p>
<p><a href="`+confirmationURL+`">Reset Password</a></p>`)

	dial := gomail.NewPlainDialer(m.Host, m.Port, m.User, m.Pass)
	return dial.DialAndSend(mail)
}
