package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Configuration holds all the confiruation for authlify
type Configuration struct {
	JWT struct {
		Secret string `json:"secret"`
		Exp    int    `json:"exp"`
	} `json:"jwt"`
	DB struct {
		Driver  string `json:"driver"`
		ConnURL string `json:"url"`
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
}

type onlyFirstLevel struct {
	Port int `json:"port"`
}

func LoadConfig(cmd *cobra.Command) (*Configuration, error) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	viper.SetEnvPrefix("NETLIFY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.netlify/authlify/")
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	config := new(onlyFirstLevel)
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	fmt.Printf("%+v\n", config)

	return nil, nil
}

// Load will construct the config from the file `config.json`
func Load() (*Configuration, error) {
	return LoadWithFile("config.json")
}

// LoadWithFile constructs the config from the specified file
func LoadWithFile(filePath string) (*Configuration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(file)

	var conf Configuration
	if err := decoder.Decode(&conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
