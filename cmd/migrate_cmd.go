package cmd

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var migrateCmd = cobra.Command{
	Use:  "migrate",
	Long: "AddMigration database strucutures. This will create new tables and add missing columns and indexes.",
	Run:  migrate,
}

func migrate(cmd *cobra.Command, args []string) {
	globalConfig, err := conf.LoadGlobal(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}
	conn, err := storage.Dial(globalConfig)
	if err != nil {
		logrus.Fatalf("%+v", errors.Wrap(err, "opening db connection"))
	}
	if err := storage.MigrateDatabase(conn); err != nil {
		logrus.Fatalf("%+v", errors.Wrap(err, "running db migrations"))
	}
	logrus.Infof("migration complete")
}
