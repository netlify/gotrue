package storage

import (
	"context"
	"time"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/netlify/gotrue/conf"
	"github.com/sirupsen/logrus"
	tconf "github.com/tigrisdata/tigris-client-go/config"
	"github.com/tigrisdata/tigris-client-go/driver"
	"github.com/tigrisdata/tigris-client-go/tigris"
)

func Client(ctx context.Context, config *conf.GlobalConfiguration) (*tigris.Client, error) {
	logrus.Infof("creating tigris driver for url: %s project: %s", config.DB.URL, config.DB.Project)
	// ToDo: project creation is not needed here, this is to create the project in the local setup.

	var drv driver.Driver
	var err error
	for i := 0; i < 3; i++ {
		drv, err = driver.NewDriver(ctx, &tconf.Driver{URL: config.DB.URL})
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
	}
	if err != nil {
		logrus.Errorf("failed in creating driver url: %s err: %v", config.DB.URL, err)
		return nil, err
	}
	logrus.Infof("creating tigris driver successful for url: %s project: %s", config.DB.URL, config.DB.Project)

	_, _ = drv.DeleteProject(ctx, config.DB.Project)
	_, err = drv.CreateProject(ctx, config.DB.Project)
	if err != nil {
		return nil, err
	}

	return tigris.NewClient(ctx, &tigris.Config{
		URL:     config.DB.URL,
		Project: config.DB.Project,
	})
}
