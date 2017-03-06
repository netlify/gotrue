package conf

import (
	"bufio"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type LogConfiguration struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

type DBConfiguration struct {
	Driver      string `json:"driver"`
	ConnURL     string `json:"url"`
	Namespace   string `json:"namespace"`
	Automigrate bool   `json:"automigrate"`
}

type JWTConfiguration struct {
	Secret             string `json:"secret"`
	Exp                int    `json:"exp"`
	AdminGroupName     string `json:"admin_group_name"`
	AdminGroupDisabled bool   `json:"admin_group_disabled"`
}

// Configuration holds all the confiruation for gotrue
type Configuration struct {
	JWT JWTConfiguration `json:"jwt"`
	DB  DBConfiguration  `json:"db"`
	API struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"api"`
	Mailer struct {
		SiteURL        string `json:"site_url"`
		Host           string `json:"host"`
		Port           int    `json:"port"`
		User           string `json:"user"`
		Pass           string `json:"pass"`
		TemplateFolder string `json:"template_folder"`
		MemberFolder   string `json:"member_folder"`
		AdminEmail     string `json:"admin_email"`
		Subjects       struct {
			Confirmation string `json:"confirmation"`
			Recovery     string `json:"recovery"`
			EmailChange  string `json:"email_change"`
		} `json:"subjects"`
		Templates struct {
			Confirmation string `json:"confirmation"`
			Recovery     string `json:"recovery"`
			EmailChange  string `json:"email_change"`
		} `json:"templates"`
	} `json:"mailer"`
	Logging LogConfiguration `json:"logging"`
}

func LoadConfig(cmd *cobra.Command) (*Configuration, error) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	viper.SetEnvPrefix("NETLIFY_AUTH")

	viper.SetConfigType("json")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.netlify/gotrue/") // keep backwards compatibility
	}

	if err := viper.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	config := new(Configuration)
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	if err := populateConfig(config); err != nil {
		return nil, err
	}

	if err := configureLogging(config); err != nil {
		return nil, err
	}

	if config.JWT.AdminGroupName == "" {
		config.JWT.AdminGroupName = "admin"
	}

	if config.JWT.Exp == 0 {
		config.JWT.Exp = 3600
	}

	return config, nil
}

func configureLogging(config *Configuration) error {
	logConfig := config.Logging

	if logConfig.File != "" {
		f, errOpen := os.OpenFile(logConfig.File, os.O_RDWR|os.O_APPEND, 0660)
		if errOpen != nil {
			return errOpen
		}
		logrus.SetOutput(bufio.NewWriter(f))
	}

	level, err := logConfig.ParseLevel()
	if err != nil {
		return err
	}
	if level != nil {
		logrus.SetLevel(*level)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableTimestamp: false,
	})

	return nil
}

func (l LogConfiguration) ParseLevel() (*logrus.Level, error) {
	if l.Level == "" {
		return nil, nil
	}

	level, err := logrus.ParseLevel(strings.ToUpper(l.Level))
	if err != nil {
		return nil, errors.Wrap(err, "parsing log level information")
	}

	return &level, nil
}

func (l LogConfiguration) IsDebugEnabled() bool {
	level, err := l.ParseLevel()
	if err != nil {
		return false
	}

	return level != nil && *level == logrus.DebugLevel
}
