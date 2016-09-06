package cmd

import (
	"fmt"
	"log"

	_ "github.com/jinzhu/gorm/dialects/sqlite" // Import SQLite
	"github.com/spf13/cobra"

	"github.com/netlify/authlify/api"
	"github.com/netlify/authlify/conf"
	"github.com/netlify/authlify/mailer"
	"github.com/netlify/authlify/models"
)

var rootCmd = cobra.Command{
	Use: "authlify",
	Run: run,
}

// RootCommand will setup and return the root command
func RootCommand() *cobra.Command {
	rootCmd.PersistentFlags().StringP("config", "c", "", "the config file to use")

	return &rootCmd
}

func run(cmd *cobra.Command, args []string) {
	config, err := conf.LoadConfig(cmd)
	if err != nil {
		log.Fatal("Failed to load config: " + err.Error())
	}

	db, err := models.Connect(config)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	mailer := mailer.NewMailer(config)

	api := api.NewAPI(config, db, mailer)

	api.ListenAndServe(fmt.Sprintf("%v:%v", config.API.Host, config.API.Port))
}
