package mailer

import (
	"bytes"
	"net/url"
	"regexp"
	"text/template"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/mailme"
	"github.com/sirupsen/logrus"
)

// Mailer defines the interface a mailer must implement.
type Mailer interface {
	Send(user *models.User, subject, body string, data map[string]interface{}) error
	InviteMail(user *models.User, referrerURL string) error
	ConfirmationMail(user *models.User, referrerURL string) error
	RecoveryMail(user *models.User, referrerURL string) error
	EmailChangeMail(user *models.User, referrerURL string) error
	ValidateEmail(email string) error
}

func interpretAsTemplate(templateString string, data map[string]interface{}) string {
	t, err := template.New("email").Parse(templateString)
	if err != nil {
		panic(err)
	}

	var result bytes.Buffer
	err = t.Execute(&result, data)
	if err != nil {
		panic(err)
	}

	return result.String()
}

func getEmailFrom(instanceConfig *conf.Configuration) string {
	return interpretAsTemplate(instanceConfig.SMTP.AdminEmail, map[string]interface{}{
		"SiteURL": instanceConfig.SiteURL,
		"SiteID":  "something", // where do we get this from?
	})
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
			From:    getEmailFrom(instanceConfig),
			BaseURL: instanceConfig.SiteURL,
			Logger:  logrus.New(),
		},
	}
}

func withDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getSiteURL(referrerURL, siteURL, filepath, fragment string) (string, error) {
	baseURL := siteURL
	if filepath == "" && referrerURL != "" {
		baseURL = referrerURL
	}

	site, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if filepath != "" {
		path, err := url.Parse(filepath)
		if err != nil {
			return "", err
		}
		site = site.ResolveReference(path)
	}
	site.Fragment = fragment
	return site.String(), nil
}

var urlRegexp = regexp.MustCompile(`^https?://[^/]+`)

func enforceRelativeURL(url string) string {
	return urlRegexp.ReplaceAllString(url, "")
}
