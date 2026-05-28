package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSettings_DefaultProviders(t *testing.T) {
	api, _, _, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	// Setup request
	req := httptest.NewRequest(http.MethodGet, "http://localhost/settings", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.handler.ServeHTTP(w, req)
	require.Equal(t, w.Code, http.StatusOK)
	resp := Settings{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	p := resp.ExternalProviders
	require.True(t, p.Email)
	require.True(t, p.Google)
	require.True(t, p.GitHub)
	require.True(t, p.GitLab)
	require.True(t, p.Bitbucket)
	require.True(t, p.SAML)
	require.True(t, p.Facebook)
}

func TestSettings_EmailDisabled(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	config.External.Email.Disabled = true

	// Setup request
	req := httptest.NewRequest(http.MethodGet, "http://localhost/settings", nil)
	req.Header.Set("Content-Type", "application/json")

	ctx, err := WithInstanceConfig(context.Background(), config, instanceID)
	require.NoError(t, err)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	api.handler.ServeHTTP(w, req)
	require.Equal(t, w.Code, http.StatusOK)
	resp := Settings{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	p := resp.ExternalProviders
	require.False(t, p.Email)
}

func TestSettings_SecurityStrictExposed(t *testing.T) {
	api, _, _, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://localhost/settings", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.handler.ServeHTTP(w, req)
	require.Equal(t, w.Code, http.StatusOK)

	strict := decodeSecurityStrict(t, w.Body)
	require.False(t, strict, "default config should report security_strict=false")
}

func TestSettings_SecurityStrictReflectsConfig(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	config.Security.Strict = true

	req := httptest.NewRequest(http.MethodGet, "http://localhost/settings", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx, err := WithInstanceConfig(context.Background(), config, instanceID)
	require.NoError(t, err)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	api.handler.ServeHTTP(w, req)
	require.Equal(t, w.Code, http.StatusOK)

	require.True(t, decodeSecurityStrict(t, w.Body))
}

// decodeSecurityStrict reads the /settings response body as a raw map so the
// test fails loudly if the security_strict key disappears from the JSON
// contract — a Settings{} decode would silently treat a missing bool as false.
func decodeSecurityStrict(t *testing.T, body io.Reader) bool {
	t.Helper()
	raw := map[string]json.RawMessage{}
	require.NoError(t, json.NewDecoder(body).Decode(&raw))
	rawStrict, ok := raw["security_strict"]
	require.True(t, ok, "settings response must include security_strict")
	var strict bool
	require.NoError(t, json.Unmarshal(rawStrict, &strict))
	return strict
}

func TestSettings_ExternalName(t *testing.T) {
	api, _, _, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://localhost/settings", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.handler.ServeHTTP(w, req)

	require.Equal(t, w.Code, http.StatusOK)

	type SettingsWithExternalName struct {
		ExternalLabels struct {
			SAML string `json:"saml"`
		} `json:"external_labels"`
	}
	resp := SettingsWithExternalName{}
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	n := resp.ExternalLabels
	require.Equal(t, n.SAML, "TestSamlName")
}
