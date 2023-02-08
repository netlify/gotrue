package cmd

import (
	"context"
	"fmt"
	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/conf"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var multiCmd = cobra.Command{
	Use:  "multi",
	Long: "Start multi-tenant API server",
	Run:  multi,
}

func multi(cmd *cobra.Command, args []string) {
	globalConfig, err := conf.LoadGlobal(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}
	if globalConfig.OperatorToken == "" {
		logrus.Fatal("Operator token secret is required")
	}

	config, err := conf.LoadConfig(configFile)
	if err != nil {
		logrus.Fatal("couldn't load config")
	}

	globalConfig.MultiInstanceMode = true
	api := api.NewAPIWithVersion(context.Background(), globalConfig, config, bootstrapSchemas(context.TODO(), globalConfig), Version)

	l := fmt.Sprintf("%v:%v", globalConfig.API.Host, globalConfig.API.Port)
	logrus.Infof("GoTrue API started on: %s", l)
	api.ListenAndServe(l)
}
