package conf

import (
	"os"
	"strconv"
	"time"

	"github.com/netlify/netlify-commons/nconf"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ExternalConfiguration holds all config related to external account providers.
type ExternalConfiguration struct {
	ClientID    string `json:"client_id"`
	Secret      string `json:"secret"`
	RedirectURI string `json:"redirect_uri"`
	URL         string `json:"url"`
}

// DBConfiguration holds all the database related configuration.
type DBConfiguration struct {
	Driver      string `json:"driver"`
	ConnURL     string `json:"url"`
	Namespace   string `json:"namespace"`
	Automigrate bool   `json:"automigrate"`
}

// JWTConfiguration holds all the JWT related configuration.
type JWTConfiguration struct {
	Secret             string `json:"secret"`
	Exp                int    `json:"exp"`
	Aud                string `json:"aud"`
	AdminGroupName     string `json:"admin_group_name"`
	AdminGroupDisabled bool   `json:"admin_group_disabled"`
	DefaultGroupName   string `json:"default_group_name"`
}

// GlobalConfiguration holds all the configuration that applies to all instances.
type GlobalConfiguration struct {
	API struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Endpoint string `json:"endpoint"`
	} `json:"api"`
	DB                DBConfiguration     `json:"db"`
	Logging           nconf.LoggingConfig `json:"log_conf"`
	NetlifySecret     string              `json:"netlify_secret"`
	MultiInstanceMode bool                `json:"-"`
}

// Configuration holds all the per-instance configuration.
type Configuration struct {
	SiteURL string           `json:"site_url"`
	JWT     JWTConfiguration `json:"jwt"`
	Mailer  struct {
		MaxFrequency time.Duration `json:"max_frequency"`
		Autoconfirm  bool          `json:"autoconfirm"`
		Host         string        `json:"host"`
		Port         int           `json:"port"`
		User         string        `json:"user"`
		Pass         string        `json:"pass"`
		MemberFolder string        `json:"member_folder"`
		AdminEmail   string        `json:"admin_email"`
		Subjects     struct {
			Invite       string `json:"invite"`
			Confirmation string `json:"confirmation"`
			Recovery     string `json:"recovery"`
			EmailChange  string `json:"email_change"`
		} `json:"subjects"`
		Templates struct {
			Invite       string `json:"invite"`
			Confirmation string `json:"confirmation"`
			Recovery     string `json:"recovery"`
			EmailChange  string `json:"email_change"`
		} `json:"templates"`
	} `json:"mailer"`
	External struct {
		Github    ExternalConfiguration `json:"github"`
		Bitbucket ExternalConfiguration `json:"bitbucket"`
		Gitlab    ExternalConfiguration `json:"gitlab"`
	} `json:"external"`
}

// LoadGlobalFromFile loads global configuration from the provided filename.
func LoadGlobalFromFile(name string) (*GlobalConfiguration, error) {
	cmd := &cobra.Command{}
	config := ""
	cmd.Flags().StringVar(&config, "config", "config.test.json", "Config file")
	cmd.Flags().Set("config", name)
	return LoadGlobal(cmd)
}

// LoadGlobal loads configuration from file and environment variables.
func LoadGlobal(cmd *cobra.Command) (*GlobalConfiguration, error) {
	config := new(GlobalConfiguration)

	if err := nconf.LoadConfig(cmd, "gotrue", config); err != nil {
		return nil, err
	}

	if config.DB.ConnURL == "" && os.Getenv("DATABASE_URL") != "" {
		config.DB.ConnURL = os.Getenv("DATABASE_URL")
	}

	if config.API.Port == 0 && os.Getenv("PORT") != "" {
		port, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return nil, errors.Wrap(err, "formatting PORT into int")
		}

		config.API.Port = port
	}

	if config.API.Port == 0 && config.API.Host == "" {
		config.API.Port = 8081
	}

	if _, err := nconf.ConfigureLogging(&config.Logging); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadConfigFromFile loads per-instance configuration from the provided filename.
func LoadConfigFromFile(name string) (*Configuration, error) {
	cmd := &cobra.Command{}
	config := ""
	cmd.Flags().StringVar(&config, "config", "config.test.json", "Config file")
	cmd.Flags().Set("config", name)
	return LoadConfig(cmd)
}

// LoadConfig loads per-instance configuration.
func LoadConfig(cmd *cobra.Command) (*Configuration, error) {
	config := new(Configuration)
	if err := nconf.LoadConfig(cmd, "gotrue", config); err != nil {
		return nil, err
	}

	config.ApplyDefaults()
	return config, nil
}

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

	if config.Mailer.MemberFolder == "" {
		config.Mailer.MemberFolder = "/member"
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
}
