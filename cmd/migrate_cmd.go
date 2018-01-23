package cmd

import (
	"fmt"
	"net/url"

	"github.com/jmoiron/sqlx"
	"github.com/markbates/pop"
	"github.com/netlify/gotrue/conf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var migrateCmd = cobra.Command{
	Use:  "migrate",
	Long: "Migrate database strucutures. This will create new tables and add missing columns and indexes.",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, migrate)
	},
}

func migrate(globalConfig *conf.GlobalConfiguration, config *conf.Configuration) {
	if globalConfig.DB.Driver == "" && globalConfig.DB.URL != "" {
		u, err := url.Parse(globalConfig.DB.URL)
		if err != nil {
			logrus.Fatalf("%+v", errors.Wrap(err, "parsing db connection url"))
		}
		globalConfig.DB.Driver = u.Scheme
	}
	pop.Debug = true

	deets := &pop.ConnectionDetails{
		Dialect: globalConfig.DB.Driver,
		URL:     globalConfig.DB.URL,
	}
	if globalConfig.DB.Namespace != "" {
		deets.Options = map[string]string{
			"Namespace": globalConfig.DB.Namespace + "_",
		}
	}

	db, err := pop.NewConnection(deets)
	if err != nil {
		logrus.Fatalf("%+v", errors.Wrap(err, "opening db connection"))
	}
	defer db.Close()

	err = createDB(db)
	if err != nil {
		logrus.Fatalf("%+v", errors.Wrap(err, "creating database"))
	}

	if err := db.Open(); err != nil {
		logrus.Fatalf("%+v", errors.Wrap(err, "checking database connection"))
	}

	mig, err := pop.NewFileMigrator(globalConfig.DB.MigrationsPath, db)
	if err != nil {
		logrus.Fatalf("%+v", errors.Wrap(err, "creating db migrator"))
	}
	// turn off schema dump
	mig.SchemaPath = ""

	err = mig.Up()
	if err != nil {
		logrus.Fatalf("%+v", errors.Wrap(err, "running db migrations"))
	}
}

func createDB(conn *pop.Connection) error {
	deets := conn.Dialect.Details()
	s := "%s:%s@(%s:%s)/?parseTime=true&multiStatements=true&readTimeout=1s"
	urlWithoutDb := fmt.Sprintf(s, deets.User, deets.Password, deets.Host, deets.Port)

	db, err := sqlx.Open(deets.Dialect, urlWithoutDb)
	if err != nil {
		return errors.Wrapf(err, "error creating MySQL database %s", deets.Database)
	}
	defer db.Close()
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT COLLATE `utf8mb4_unicode_ci`", deets.Database)

	_, err = db.Exec(query)
	if err != nil {
		return errors.Wrapf(err, "error creating MySQL database %s", deets.Database)
	}
	return nil
}
