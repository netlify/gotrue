package api

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// TestCORS_FlagOffPreservesWildcard guards the core safety property: when an
// instance has not opted in, the CORS response is byte-for-byte the historical
// default (Access-Control-Allow-Origin: *), not a reflected origin. Runs
// without a database because preflight handling never reaches the router.
func TestCORS_FlagOffPreservesWildcard(t *testing.T) {
	config := &conf.Configuration{}
	config.ApplyDefaults()
	ctx, err := WithInstanceConfig(context.Background(), config, uuid.Nil)
	require.NoError(t, err)
	api := NewAPIWithVersion(ctx, &conf.GlobalConfiguration{}, nil, "test")

	req := httptest.NewRequest(http.MethodOptions, "/settings", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	api.handler.ServeHTTP(w, req)

	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORS_FlagOnRestrictsOrigin verifies that with Security.Enabled the
// allowlist is enforced: a non-listed origin gets no Allow-Origin header, and
// the SiteURL origin is reflected.
func TestCORS_FlagOnRestrictsOrigin(t *testing.T) {
	config := &conf.Configuration{SiteURL: "https://app.example.com"}
	config.Security.Enabled = true
	config.ApplyDefaults()
	ctx, err := WithInstanceConfig(context.Background(), config, uuid.Nil)
	require.NoError(t, err)
	api := NewAPIWithVersion(ctx, &conf.GlobalConfiguration{}, nil, "test")

	disallowed := httptest.NewRequest(http.MethodOptions, "/settings", nil)
	disallowed.Header.Set("Origin", "https://evil.example.com")
	disallowed.Header.Set("Access-Control-Request-Method", "GET")
	wd := httptest.NewRecorder()
	api.handler.ServeHTTP(wd, disallowed)
	require.Empty(t, wd.Header().Get("Access-Control-Allow-Origin"))

	allowed := httptest.NewRequest(http.MethodOptions, "/settings", nil)
	allowed.Header.Set("Origin", "https://app.example.com")
	allowed.Header.Set("Access-Control-Request-Method", "GET")
	wa := httptest.NewRecorder()
	api.handler.ServeHTTP(wa, allowed)
	require.Equal(t, "https://app.example.com", wa.Header().Get("Access-Control-Allow-Origin"))
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
