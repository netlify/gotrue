package test

import (
	"fmt"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/netlify/gotrue/storage/sql"
)

const (
	apiTestEnv = "../hack/test.env"
)

var conn storage.Connection

func SetupDBConnection() (*conf.GlobalConfiguration, storage.Connection, error) {
	globalConfig, err := conf.LoadGlobal(apiTestEnv)
	if err != nil {
		return nil, nil, err
	}

	conn, err = dial.Dial(globalConfig)
	if err != nil {
		return nil, nil, err
	}

	return globalConfig, conn, err
}

func CleanupTables() error {
	sconn, ok := conn.(*sql.Connection)
	if !ok {
		return fmt.Errorf("sql connection required for testing")
	}

	sconn.TruncateAll()

	return nil
}
