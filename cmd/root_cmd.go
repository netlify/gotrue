package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/netlify/gotrue/conf"
)

var configFile = ""

var rootCmd = cobra.Command{
	Use: "gotrue",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

// RootCommand will setup and return the root command
func RootCommand() *cobra.Command {
	rootCmd.AddCommand(&serveCmd, &migrateCmd, &multiCmd, &versionCmd, adminCmd())
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "the config file to use")

	return &rootCmd
}

func execWithConfig(cmd *cobra.Command, fn func(globalConfig *conf.GlobalConfiguration, config *conf.Configuration)) {
	globalConfig, err := conf.LoadGlobal(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}

	fn(globalConfig, config)
}

func execWithConfigAndArgs(cmd *cobra.Command, fn func(globalConfig *conf.GlobalConfiguration, config *conf.Configuration, args []string), args []string) {
	globalConfig, err := conf.LoadGlobal(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}
	config, err := conf.LoadConfig(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}

	fn(globalConfig, config, args)
}
