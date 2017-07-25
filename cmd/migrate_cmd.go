package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/spf13/cobra"
)

var migrateCmd = cobra.Command{
	Use:  "migrate",
	Long: "Migrate database strucutures. This will create new tables and add missing columns and indexes.",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, migrate)
	},
}

func migrate(config *conf.Configuration) {
	db, err := dial.Dial(config)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}

	if err := db.Automigrate(); err != nil {
		logrus.Fatalf("Error migrating tables: %+v", err)
	}
}
