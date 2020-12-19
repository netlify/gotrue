package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/api"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/rpc/servers"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var serveCmd = cobra.Command{
	Use:  "serve",
	Long: "Start API server",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

func serve(globalConfig *conf.GlobalConfiguration, config *conf.Configuration) {
	ctx, err := api.WithInstanceConfig(context.Background(), config, uuid.Nil)
	if err != nil {
		logrus.Fatalf("Error loading instance config: %+v", err)
	}
	listenAndServe(ctx, globalConfig)
}

// listenAndServe starts the API servers
func listenAndServe(ctx context.Context, globalConfig *conf.GlobalConfiguration) {

	db := openDB(globalConfig)
	a := api.NewAPIWithVersion(ctx, globalConfig, db, Version)
	log := logrus.WithField("component", "api")

	done := make(chan struct{})
	defer close(done)
	go func() {
		addr := fmt.Sprintf("%v:%v", globalConfig.API.Host, globalConfig.API.RestPort)
		logrus.Infof("GoTrue REST API started on: %s", addr)
		a.ListenAndServeREST(addr)
	}()

	servers.ListenAndServeRPC(a, globalConfig)

	util.WaitForTermination(log, done)

	log.Info("shutting down...")
}

func openDB(globalConfig *conf.GlobalConfiguration) (db *storage.Connection) {
	// try a couple times to connect to the database
	var err error
	for i := 1; i <= 3; i++ {
		time.Sleep(time.Duration((i-1)*100) * time.Millisecond)
		db, err = storage.Dial(globalConfig)
		if err == nil {
			break
		}
		logrus.WithError(err).WithField("attempt", i).Warn("Error connecting to database")
	}
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}
	return
}
