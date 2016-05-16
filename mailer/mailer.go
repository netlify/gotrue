package mailer

import (
	"log"

	"gopkg.in/gomail.v2"

	"github.com/netlify/authlify/conf"
	"github.com/netlify/authlify/models"
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
}

// NewMailer returns a new authlify mailer
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
<p><a href="`+confirmationURL+`">Complete Registration</a></p>"`)

	log.Printf("Sending from %v", m.AdminEmail)

	dial := gomail.NewDialer(m.Host, m.Port, m.User, m.Pass)
	log.Printf("Dialing to %v", dial)

	return dial.DialAndSend(mail)
}
