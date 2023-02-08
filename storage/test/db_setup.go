package test

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"context"
)

func SetupDBConnection(globalConfig *conf.GlobalConfiguration) (*tigris.Client, error) {
	return storage.Client(context.TODO(), globalConfig)
}
