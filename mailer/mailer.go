package mailer

import (
	"net/url"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/mailme"
)

// Mailer defines the interface a mailer must implement.
type Mailer interface {
	Send(user *models.User, subject, body string, data map[string]interface{}) error
	InviteMail(user *models.User) error
	ConfirmationMail(user *models.User) error
	RecoveryMail(user *models.User) error
	EmailChangeMail(user *models.User) error
	ValidateEmail(email string) error
}

// NewMailer returns a new gotrue mailer
func NewMailer(instanceConfig *conf.Configuration) Mailer {
	if instanceConfig.SMTP.Host == "" {
		return &noopMailer{}
	}

	return &TemplateMailer{
		SiteURL: instanceConfig.SiteURL,
		Config:  instanceConfig,
		Mailer: &mailme.Mailer{
			Host:    instanceConfig.SMTP.Host,
			Port:    instanceConfig.SMTP.Port,
			User:    instanceConfig.SMTP.User,
			Pass:    instanceConfig.SMTP.Pass,
			From:    instanceConfig.SMTP.AdminEmail,
			BaseURL: instanceConfig.SiteURL,
		},
	}
}

func withDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
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
