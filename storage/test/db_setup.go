package test

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
)

func SetupDBConnection(globalConfig *conf.GlobalConfiguration) (*storage.Connection, error) {
	return storage.Dial(globalConfig)
}
