package api

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
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
	cb := func(gc *conf.GlobalConfiguration, c *conf.Configuration, conn *storage.Connection) (uuid.UUID, error) {
		gc.MultiInstanceMode = true
		return uuid.Nil, nil
	}

	return setupAPIForTestWithCallback(cb)
}

func setupAPIForTestForInstance() (*API, *conf.Configuration, uuid.UUID, error) {
	instanceID := uuid.Must(uuid.NewV4())
	cb := func(gc *conf.GlobalConfiguration, c *conf.Configuration, conn *storage.Connection) (uuid.UUID, error) {
		err := conn.Create(&models.Instance{
			ID:         instanceID,
			UUID:       testUUID,
			BaseConfig: c,
		})
		return instanceID, err
	}

	api, conf, err := setupAPIForTestWithCallback(cb)
	if err != nil {
		return nil, nil, uuid.Nil, err
	}
	return api, conf, instanceID, nil
}

func setupAPIForTestWithCallback(cb func(*conf.GlobalConfiguration, *conf.Configuration, *storage.Connection) (uuid.UUID, error)) (*API, *conf.Configuration, error) {
	globalConfig, err := conf.LoadGlobal(apiTestConfig)
	if err != nil {
		return nil, nil, err
	}

	conn, err := test.SetupDBConnection(globalConfig)
	if err != nil {
		return nil, nil, err
	}

	config, err := conf.LoadConfig(apiTestConfig)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	instanceID := uuid.Nil
	if cb != nil {
		instanceID, err = cb(globalConfig, config, conn)
		if err != nil {
			conn.Close()
			return nil, nil, err
		}
	}

	ctx, err := WithInstanceConfig(context.Background(), config, instanceID)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return NewAPIWithVersion(ctx, globalConfig, conn, apiTestVersion), config, nil
}

func TestEmailEnabledByDefault(t *testing.T) {
	api, _, err := setupAPIForTest()
	require.NoError(t, err)

	require.False(t, api.config.External.Email.Disabled)
}

func TestOriginAllowed(t *testing.T) {
	cases := []struct {
		name    string
		config  *conf.Configuration
		origin  string
		allowed bool
	}{
		{
			name:    "matches configured allowlist",
			config:  &conf.Configuration{Security: conf.SecurityConfiguration{AllowedCORSOrigins: []string{"https://app.example.com"}}},
			origin:  "https://app.example.com",
			allowed: true,
		},
		{
			name:    "rejects non-listed origin",
			config:  &conf.Configuration{Security: conf.SecurityConfiguration{AllowedCORSOrigins: []string{"https://app.example.com"}}},
			origin:  "https://evil.com",
			allowed: false,
		},
		{
			name:    "case-insensitive match",
			config:  &conf.Configuration{Security: conf.SecurityConfiguration{AllowedCORSOrigins: []string{"https://APP.example.com"}}},
			origin:  "https://app.example.com",
			allowed: true,
		},
		{
			name:    "empty allowlist falls back to SiteURL origin",
			config:  &conf.Configuration{SiteURL: "https://app.example.com/some/path"},
			origin:  "https://app.example.com",
			allowed: true,
		},
		{
			name:    "empty allowlist rejects non-SiteURL origin",
			config:  &conf.Configuration{SiteURL: "https://app.example.com"},
			origin:  "https://other.example.com",
			allowed: false,
		},
		{
			name:    "empty allowlist with invalid SiteURL rejects all",
			config:  &conf.Configuration{SiteURL: "not-a-url"},
			origin:  "https://app.example.com",
			allowed: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.allowed, originAllowed(tc.config, tc.origin))
		})
	}
}
