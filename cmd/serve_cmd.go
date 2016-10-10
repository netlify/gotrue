package cmd

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/netlify/netlify-auth/api"
	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/mailer"
	"github.com/netlify/netlify-auth/models"
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
	db, err := models.Connect(config)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}

	mailer := mailer.NewMailer(config)
	api := api.NewAPIWithVersion(config, db, mailer, Version)

	l := fmt.Sprintf("%v:%v", config.API.Host, config.API.Port)
	logrus.Infof("Netlify Auth API started on: %s", l)
	api.ListenAndServe(l)
}
