package cmd

import (
	"net/url"
	"os"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/pop/v5/logging"
	"github.com/netlify/gotrue/conf"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var migrateCmd = cobra.Command{
	Use:  "migrate",
	Long: "Migrate database strucutures. This will create new tables and add missing columns and indexes.",
	Run:  migrate,
}

func migrate(cmd *cobra.Command, args []string) {
	globalConfig, err := conf.LoadGlobal(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %+v", err)
	}
	if globalConfig.DB.Driver == "" && globalConfig.DB.URL != "" {
		u, err := url.Parse(globalConfig.DB.URL)
		if err != nil {
			logrus.Fatalf("%+v", errors.Wrap(err, "parsing db connection url"))
		}
		globalConfig.DB.Driver = u.Scheme
	}

	log := logrus.New()

	pop.Debug = false
	if globalConfig.Logging.Level != "" {
		level, err := logrus.ParseLevel(globalConfig.Logging.Level)
		if err != nil {
			log.Fatalf("Failed to parse log level: %+v", err)
		}
		log.SetLevel(level)
		if level == logrus.DebugLevel {
			// Set to true to display query info
			pop.Debug = true
		}
		if level != logrus.DebugLevel {
			var noopLogger = func(lvl logging.Level, s string, args ...interface{}) {
				return
			}
			// Hide pop migration logging
			pop.SetLogger(noopLogger)
		}
	}

	deets := &pop.ConnectionDetails{
		Dialect: globalConfig.DB.Driver,
		URL:     globalConfig.DB.URL,
	}
	deets.Options = map[string]string{
		"migration_table_name": "schema_migrations",
	}

	db, err := pop.NewConnection(deets)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "opening db connection"))
	}
	defer db.Close()

	if err := db.Open(); err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "checking database connection"))
	}

	log.Debugf("Reading migrations from %s", globalConfig.DB.MigrationsPath)
	mig, err := pop.NewFileMigrator(globalConfig.DB.MigrationsPath, db)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "creating db migrator"))
	}
	log.Debugf("before status")

	if log.Level == logrus.DebugLevel {
		err = mig.Status(os.Stdout)
		if err != nil {
			log.Fatalf("%+v", errors.Wrap(err, "migration status"))
		}
	}

	// turn off schema dump
	mig.SchemaPath = ""

	err = mig.Up()
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "running db migrations"))
	}

	log.Debugf("after status")

	if log.Level == logrus.DebugLevel {
		err = mig.Status(os.Stdout)
		if err != nil {
			log.Fatalf("%+v", errors.Wrap(err, "migration status"))
		}
	}
}
