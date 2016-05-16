package mailer

import (
	"net/url"

	"gopkg.in/gomail.v2"

	"github.com/netlify/authlify/models"
)

// Mailer will send mail and use templates from the site for easy mail styling
type Mailer struct {
	SiteURL        url.URL
	TemplateFolder string
	MemberFolder   string
	Host           string
	Port           int
	User           string
	Pass           string
	AdminEmail     string
	MailSubjects   struct {
		ConfirmationMail string
	}
}

// ConfirmationMail sends a signup confirmation mail to a new user
func (m *Mailer) ConfirmationMail(user *models.User) error {
	confirmationURL := m.SiteURL.String() + m.MemberFolder + "/confirm/" + user.ConfirmationToken

	mail := gomail.NewMessage()
	mail.SetHeader("From", m.AdminEmail)
	mail.SetHeader("To", user.Email)
	mail.SetHeader("Subject", m.MailSubjects.ConfirmationMail)
	mail.SetBody("text/html", `<h2>Please verify your registration</h2>

<p>Follow this link to complete the registration process:</p>
<p><a href="`+confirmationURL+`>Complete Registration</a></p>"`)

	dial := gomail.NewDialer(m.Host, m.Port, m.User, m.Pass)

	return dial.DialAndSend(mail)
}
