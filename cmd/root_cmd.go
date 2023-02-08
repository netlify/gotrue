package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/netlify/gotrue/conf"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"github.com/netlify/gotrue/models"
	"fmt"
	"context"
	"github.com/netlify/gotrue/storage"
)

var configFile = ""

var rootCmd = cobra.Command{
	Use: "gotrue",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

var TigrisConfig = &tigris.Config{URL: fmt.Sprintf("%v:%d", "localhost", 8081), Project: "gotrue"}

// RootCommand will setup and return the root command
func RootCommand() *cobra.Command {
	rootCmd.AddCommand(&serveCmd, &multiCmd, &versionCmd, adminCmd())
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "the config file to use")

	return &rootCmd
}

func execWithConfig(cmd *cobra.Command, fn func(globalConfig *conf.GlobalConfiguration, config *conf.Configuration, database *tigris.Database)) {
	globalConfig, err := conf.LoadGlobal(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}

	db := bootstrapSchemas(context.TODO(), globalConfig)

	fn(globalConfig, config, db)
}

func execWithConfigAndArgs(cmd *cobra.Command, fn func(globalConfig *conf.GlobalConfiguration, config *conf.Configuration, database *tigris.Database, args []string), args []string) {
	globalConfig, err := conf.LoadGlobal(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}

	db := bootstrapSchemas(context.TODO(), globalConfig)

	fn(globalConfig, config, db, args)
}

func bootstrapSchemas(ctx context.Context, globalConfig *conf.GlobalConfiguration) *tigris.Database {
	tigrisClient, err := storage.Client(ctx, globalConfig)
	if err != nil {
		logrus.Fatalf("Failed to create tigris client: %+v", err)
	}

	db, err := tigrisClient.OpenDatabase(ctx, &models.AuditLogEntry{}, &models.User{}, &models.RefreshToken{}, &models.Instance{})
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}

	return db
}
