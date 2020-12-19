package cmd

import (
	"context"

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
	globalConfig.MultiInstanceMode = true
	listenAndServe(context.Background(), globalConfig)
}
