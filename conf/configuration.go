package conf

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// OAuthProviderConfiguration holds all config related to external account providers.
type OAuthProviderConfiguration struct {
	ClientID    string `json:"client_id" split_words:"true"`
	Secret      string `json:"secret"`
	RedirectURI string `json:"redirect_uri" split_words:"true"`
	URL         string `json:"url"`
	Enabled     bool   `json:"enabled"`
}

type EmailProviderConfiguration struct {
	Disabled bool `json:"disabled"`
}

type SamlProviderConfiguration struct {
	Enabled     bool   `json:"enabled"`
	MetadataURL string `json:"metadata_url" envconfig:"METADATA_URL"`
	APIBase     string `json:"api_base" envconfig:"API_BASE"`
	Name        string `json:"name"`
	SigningCert string `json:"signing_cert" envconfig:"SIGNING_CERT"`
	SigningKey  string `json:"signing_key" envconfig:"SIGNING_KEY"`
}

// DBConfiguration holds all the database related configuration.
type DBConfiguration struct {
	Driver         string `json:"driver" required:"true"`
	URL            string `json:"url" envconfig:"DATABASE_URL" required:"true"`
	Namespace      string `json:"namespace"`
	MigrationsPath string `json:"migrations_path" split_words:"true" default:"./migrations"`
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
		Host            string
		Port            int `envconfig:"PORT" default:"8081"`
		Endpoint        string
		RequestIDHeader string `envconfig:"REQUEST_ID_HEADER"`
	}
	DB                DBConfiguration
	External          ProviderConfiguration
	Logging           LoggingConfig `envconfig:"LOG"`
	OperatorToken     string        `split_words:"true" required:"true"`
	MultiInstanceMode bool
	Tracing           TracingConfig
	SMTP              SMTPConfiguration
	RateLimitHeader   string `split_words:"true"`
}

// EmailContentConfiguration holds the configuration for emails, both subjects and template URLs.
type EmailContentConfiguration struct {
	Invite       string `json:"invite"`
	Confirmation string `json:"confirmation"`
	Recovery     string `json:"recovery"`
	EmailChange  string `json:"email_change" split_words:"true"`
}

type ProviderConfiguration struct {
	Bitbucket   OAuthProviderConfiguration `json:"bitbucket"`
	Github      OAuthProviderConfiguration `json:"github"`
	Gitlab      OAuthProviderConfiguration `json:"gitlab"`
	Google      OAuthProviderConfiguration `json:"google"`
	Facebook    OAuthProviderConfiguration `json:"facebook"`
	Email       EmailProviderConfiguration `json:"email"`
	Saml        SamlProviderConfiguration  `json:"saml"`
	RedirectURL string                     `json:"redirect_url"`
}

type SMTPConfiguration struct {
	MaxFrequency time.Duration `json:"max_frequency" split_words:"true"`
	Host         string        `json:"host"`
	Port         int           `json:"port,omitempty" default:"587"`
	User         string        `json:"user"`
	Pass         string        `json:"pass,omitempty"`
	AdminEmail   string        `json:"admin_email" split_words:"true"`
}

type MailerConfiguration struct {
	Autoconfirm bool                      `json:"autoconfirm"`
	Subjects    EmailContentConfiguration `json:"subjects"`
	Templates   EmailContentConfiguration `json:"templates"`
	URLPaths    EmailContentConfiguration `json:"url_paths"`
}

// Configuration holds all the per-instance configuration.
type Configuration struct {
	SiteURL       string                `json:"site_url" split_words:"true" required:"true"`
	JWT           JWTConfiguration      `json:"jwt"`
	SMTP          SMTPConfiguration     `json:"smtp"`
	Mailer        MailerConfiguration   `json:"mailer"`
	External      ProviderConfiguration `json:"external"`
	DisableSignup bool                  `json:"disable_signup" split_words:"true"`
	Webhook       WebhookConfig         `json:"webhook" split_words:"true"`
	Cookie        struct {
		Key      string `json:"key"`
		Duration int    `json:"duration"`
	} `json:"cookies"`
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

type WebhookConfig struct {
	URL        string   `json:"url"`
	Retries    int      `json:"retries"`
	TimeoutSec int      `json:"timeout_sec"`
	Secret     string   `json:"secret"`
	Events     []string `json:"events"`
}

func (w *WebhookConfig) HasEvent(event string) bool {
	for _, name := range w.Events {
		if event == name {
			return true
		}
	}
	return false
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

	if _, err := ConfigureLogging(&config.Logging); err != nil {
		return nil, err
	}

	ConfigureTracing(&config.Tracing)

	if config.SMTP.MaxFrequency == 0 {
		config.SMTP.MaxFrequency = 15 * time.Minute
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

	if config.SMTP.MaxFrequency == 0 {
		config.SMTP.MaxFrequency = 15 * time.Minute
	}

	if config.Cookie.Key == "" {
		config.Cookie.Key = "nf_jwt"
	}

	if config.Cookie.Duration == 0 {
		config.Cookie.Duration = 86400
	}
}

func (config *Configuration) Value() (driver.Value, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return driver.Value(""), err
	}
	return driver.Value(string(data)), nil
}

func (config *Configuration) Scan(src interface{}) error {
	var source []byte
	switch v := src.(type) {
	case string:
		source = []byte(v)
	case []byte:
		source = v
	default:
		return errors.New("Invalid data type for Configuration")
	}

	if len(source) == 0 {
		source = []byte("{}")
	}
	return json.Unmarshal(source, &config)
}

func (o *OAuthProviderConfiguration) Validate() error {
	if !o.Enabled {
		return errors.New("Provider is not enabled")
	}
	if o.ClientID == "" {
		return errors.New("Missing Oauth client ID")
	}
	if o.Secret == "" {
		return errors.New("Missing Oauth secret")
	}
	if o.RedirectURI == "" {
		return errors.New("Missing redirect URI")
	}
	return nil
}
