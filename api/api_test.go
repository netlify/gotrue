package api

import (
	"context"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/pborman/uuid"
)

const (
	apiTestVersion = "1"
	apiTestConfig  = "../hack/test.env"
)

// setupAPIForTest creates a new API to run tests with.
// Using this function allows us to keep track of the database connection
// and cleaning up data between tests.
func setupAPIForTest() (*API, *conf.Configuration, error) {
	return setupAPIForTestWithCallback(nil)
}

func setupAPIForMultiinstanceTest() (*API, *conf.Configuration, error) {
	cb := func(gc *conf.GlobalConfiguration, c *conf.Configuration, conn storage.Connection) (string, error) {
		gc.MultiInstanceMode = true
		return "", nil
	}

	return setupAPIForTestWithCallback(cb)
}

func setupAPIForTestForInstance() (*API, *conf.Configuration, string, error) {
	instanceID := uuid.NewRandom().String()
	cb := func(gc *conf.GlobalConfiguration, c *conf.Configuration, conn storage.Connection) (string, error) {
		err := conn.CreateInstance(&models.Instance{
			ID:         instanceID,
			UUID:       testUUID,
			BaseConfig: c,
		})
		return instanceID, err
	}

	api, conf, err := setupAPIForTestWithCallback(cb)
	if err != nil {
		return nil, nil, "", err
	}
	return api, conf, instanceID, nil
}

func setupAPIForTestWithCallback(cb func(*conf.GlobalConfiguration, *conf.Configuration, storage.Connection) (string, error)) (*API, *conf.Configuration, error) {
	globalConfig, conn, err := test.SetupDBConnection()
	if err != nil {
		return nil, nil, err
	}

	config, err := conf.LoadConfig(apiTestConfig)
	if err != nil {
		return nil, nil, err
	}

	instanceID := ""
	if cb != nil {
		instanceID, err = cb(globalConfig, config, conn)
		if err != nil {
			return nil, nil, err
		}
	}

	ctx, err := WithInstanceConfig(context.Background(), globalConfig.SMTP, config, instanceID)
	if err != nil {
		return nil, nil, err
	}

	return NewAPIWithVersion(ctx, globalConfig, conn, apiTestVersion), config, nil
}
