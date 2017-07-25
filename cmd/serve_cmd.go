package cmd

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/mailer"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/spf13/cobra"
)

var serveCmd = cobra.Command{
	Use:  "serve",
	Long: "Start API server",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

func serve(config *conf.Configuration) {
	db, err := dial.Dial(config)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}
	defer db.Close()

	if config.DB.Automigrate {
		if err := db.Automigrate(); err != nil {
			logrus.Fatalf("Error migrating models: %+v", err)
		}
	}

	mailer := mailer.NewMailer(config)
	api := api.NewAPIWithVersion(config, db, mailer, Version)

	l := fmt.Sprintf("%v:%v", config.API.Host, config.API.Port)
	logrus.Infof("GoTrue API started on: %s", l)
	api.ListenAndServe(l)
}
