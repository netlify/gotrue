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

// Configuration holds all the confiruation for netlify-auth
type Configuration struct {
	JWT struct {
		Secret string `json:"secret"`
		Exp    int    `json:"exp"`
	} `json:"jwt"`
	DB struct {
		Driver    string `json:"driver"`
		ConnURL   string `json:"url"`
		Namespace string `json:"namespace"`
	}
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
		MailSubjects   struct {
			ConfirmationMail string `json:"confirmation"`
			RecoveryMail     string `json:"recovery"`
		} `json:"mail_subjects"`
	} `json:"mailer"`
	Logging struct {
		Level string `json:"level"`
		File  string `json:"file"`
	} `json:"logging"`
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
		viper.AddConfigPath("$HOME/.netlify/authlify/") // keep backwards compatibility
		viper.AddConfigPath("$HOME/.netlify/netlify-auth/")
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

	if logConfig.Level != "" {
		level, err := logrus.ParseLevel(strings.ToUpper(logConfig.Level))
		if err != nil {
			return errors.Wrap(err, "configuring logging")
		}
		logrus.SetLevel(level)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableTimestamp: false,
	})

	return nil
}
