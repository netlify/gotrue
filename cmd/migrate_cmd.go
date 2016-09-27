package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/models"
	"github.com/spf13/cobra"
)

var migrateCmd = cobra.Command{
	Use:  "migrate",
	Long: "Migrate database strucutures. This will create new tables and add missing collumns and indexes.",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, migrate)
	},
}

func migrate(config *conf.Configuration) {
	db, err := models.Connect(config)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}

	db.AutoMigrate(models.RefreshToken{})
	db.AutoMigrate(models.User{})
	db.AutoMigrate(models.Data{})
}
