package cmd

import (
	"context"
	"fmt"

	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var multiCmd = cobra.Command{
	Use:  "multi",
	Long: "Start multi-tenant API server",
	Run:  multi,
}

func multi(cmd *cobra.Command, args []string) {
	globalConfig, err := conf.LoadGlobal(cmd)
	if err != nil {
		logrus.Fatalf("Failed to load configration: %+v", err)
	}

	db, err := dial.Dial(globalConfig)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}
	defer db.Close()

	if globalConfig.DB.Automigrate {
		if err := db.Automigrate(); err != nil {
			logrus.Fatalf("Error migrating models: %+v", err)
		}
	}

	globalConfig.MultiInstanceMode = true
	api := api.NewAPIWithVersion(context.Background(), globalConfig, db, Version)

	l := fmt.Sprintf("%v:%v", globalConfig.API.Host, globalConfig.API.Port)
	logrus.Infof("GoTrue API started on: %s", l)
	api.ListenAndServe(l)
}
