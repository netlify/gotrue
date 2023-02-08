package storage

import (
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/netlify/gotrue/conf"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"context"
	tconf "github.com/tigrisdata/tigris-client-go/config"
	"github.com/tigrisdata/tigris-client-go/driver"
)

func Client(ctx context.Context, config *conf.GlobalConfiguration) (*tigris.Client, error) {
	// ToDo: project creation is not needed here, this is to create the project in the local setup.
	drv, _ := driver.NewDriver(ctx, &tconf.Driver{URL: config.DB.URL})
	_, _ = drv.DeleteProject(ctx, config.DB.Project)
	_, err := drv.CreateProject(ctx, config.DB.Project)
	if err != nil {
		return nil, err
	}

	return tigris.NewClient(ctx, &tigris.Config{
		URL:     config.DB.URL,
		Project: config.DB.Project,
	})
}
