package conf

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/netlify/netlify-commons/nconf"
)

// OAuthProviderConfiguration holds all config related to external account providers.
type OAuthProviderConfiguration struct {
	ClientID    string `json:"client_id" split_words:"true"`
	Secret      string `json:"secret"`
	RedirectURI string `json:"redirect_uri" split_words:"true"`
	URL         string `json:"url"`
}

// DBConfiguration holds all the database related configuration.
type DBConfiguration struct {
	Dialect     string `json:"dialect"`
	Driver      string `json:"driver" required:"true"`
	URL         string `json:"url" envconfig:"DATABASE_URL" required:"true"`
	Namespace   string `json:"namespace"`
	Automigrate bool   `json:"automigrate"`
}

// JWTConfiguration holds all the JWT related configuration.
type JWTConfiguration struct {
	Secret           string `json:"secret" required:"true"`
	Exp              int    `json:"exp"`
	Aud              string `json:"aud"`
	AdminGroupName   string `json:"admin_group_name" split_words:"true"`
	DefaultGroupName string `json:"default_group_name" split_words:"true"`
}

// GlobalConfiguration holds all the configuration that applies to all instances.
type GlobalConfiguration struct {
	API struct {
		Host     string
		Port     int `envconfig:"PORT" default:"8081"`
		Endpoint string
	}
	DB                DBConfiguration
	External          ExternalProviderConfiguration
	Logging           nconf.LoggingConfig `envconfig:"LOG"`
	OperatorToken     string              `split_words:"true"`
	MultiInstanceMode bool
}

// EmailContentConfiguration holds the configuration for emails, both subjects and template URLs.
type EmailContentConfiguration struct {
	Invite       string `json:"invite"`
	Confirmation string `json:"confirmation"`
	Recovery     string `json:"recovery"`
	EmailChange  string `json:"email_change" split_words:"true"`
}

type ExternalProviderConfiguration struct {
	Bitbucket   OAuthProviderConfiguration `json:"bitbucket"`
	Github      OAuthProviderConfiguration `json:"github"`
	Gitlab      OAuthProviderConfiguration `json:"gitlab"`
	Google      OAuthProviderConfiguration `json:"google"`
	RedirectURL string                     `json:"redirect_url"`
}

// Configuration holds all the per-instance configuration.
type Configuration struct {
	SiteURL string           `json:"site_url" split_words:"true" required:"true"`
	JWT     JWTConfiguration `json:"jwt"`
	Mailer  struct {
		MaxFrequency time.Duration             `json:"max_frequency" split_words:"true"`
		Autoconfirm  bool                      `json:"autoconfirm"`
		Host         string                    `json:"host"`
		Port         int                       `json:"port"`
		User         string                    `json:"user"`
		Pass         string                    `json:"pass"`
		AdminEmail   string                    `json:"admin_email" split_words:"true"`
		Subjects     EmailContentConfiguration `json:"subjects"`
		Templates    EmailContentConfiguration `json:"templates"`
		URLPaths     EmailContentConfiguration `json:"url_paths"`
	} `json:"mailer"`
	External ExternalProviderConfiguration `json:"external"`
}

func loadEnvironment(filename string) error {
	var err error
	if filename != "" {
		err = godotenv.Load(filename)
	} else {
		err = godotenv.Load()
		// handle if .env file does not exist, this is OK
		if os.IsNotExist(err) {
			return nil
		}
	}
	return err
}

// LoadGlobal loads configuration from file and environment variables.
func LoadGlobal(filename string) (*GlobalConfiguration, error) {
	if err := loadEnvironment(filename); err != nil {
		return nil, err
	}

	config := new(GlobalConfiguration)
	if err := envconfig.Process("gotrue", config); err != nil {
		return nil, err
	}
	if _, err := nconf.ConfigureLogging(&config.Logging); err != nil {
		return nil, err
	}
	return config, nil
}

// LoadConfig loads per-instance configuration.
func LoadConfig(filename string) (*Configuration, error) {
	if err := loadEnvironment(filename); err != nil {
		return nil, err
	}

	config := new(Configuration)
	if err := envconfig.Process("gotrue", config); err != nil {
		return nil, err
	}
	config.ApplyDefaults()
	return config, nil
}

// ApplyDefaults sets defaults for a Configuration
func (config *Configuration) ApplyDefaults() {
	if config.JWT.AdminGroupName == "" {
		config.JWT.AdminGroupName = "admin"
	}

	if config.JWT.Exp == 0 {
		config.JWT.Exp = 3600
	}

	if config.Mailer.MaxFrequency == 0 {
		config.Mailer.MaxFrequency = 15 * time.Minute
	}

	if config.Mailer.Templates.Invite == "" {
		config.Mailer.Templates.Invite = "/.netlify/gotrue/templates/invite.html"
	}
	if config.Mailer.Templates.Confirmation == "" {
		config.Mailer.Templates.Confirmation = "/.netlify/gotrue/templates/confirm.html"
	}
	if config.Mailer.Templates.Recovery == "" {
		config.Mailer.Templates.Recovery = "/.netlify/gotrue/templates/recover.html"
	}
	if config.Mailer.Templates.EmailChange == "" {
		config.Mailer.Templates.EmailChange = "/.netlify/gotrue/templates/email-change.html"
	}

	if config.Mailer.URLPaths.Invite == "" {
		config.Mailer.URLPaths.Invite = "/"
	}
	if config.Mailer.URLPaths.Confirmation == "" {
		config.Mailer.URLPaths.Confirmation = "/"
	}
	if config.Mailer.URLPaths.Recovery == "" {
		config.Mailer.URLPaths.Recovery = "/"
	}
	if config.Mailer.URLPaths.EmailChange == "" {
		config.Mailer.URLPaths.EmailChange = "/"
	}
}
