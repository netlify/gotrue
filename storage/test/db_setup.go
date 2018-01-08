package test

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/dial"
)

func SetupDBConnection(globalConfig *conf.GlobalConfiguration) (storage.Connection, error) {
	return dial.Dial(globalConfig)
}
