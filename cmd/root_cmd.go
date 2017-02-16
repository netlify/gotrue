package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/netlify/netlify-auth/conf"
)

var rootCmd = cobra.Command{
	Use: "netlify-auth",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

// RootCommand will setup and return the root command
func RootCommand() *cobra.Command {
	rootCmd.AddCommand(&serveCmd, &migrateCmd, &versionCmd)
	rootCmd.PersistentFlags().StringP("config", "c", "", "the config file to use")

	return &rootCmd
}

func execWithConfig(cmd *cobra.Command, fn func(config *conf.Configuration)) {
	config, err := conf.LoadConfig(cmd)
	if err != nil {
		logrus.Fatalf("Failed to load configration: %+v", err)
	}

	fn(config)
}
