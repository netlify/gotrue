package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netlify/gotrue/conf"
)

func TestSetCookieTokenSameSite(t *testing.T) {
	config := &conf.Configuration{}
	config.Cookie.Key = "nf_jwt"
	config.Cookie.Duration = 3600

	for _, session := range []bool{true, false} {
		w := httptest.NewRecorder()
		api := &API{}
		require.NoError(t, api.setCookieToken(config, "the-token", session, w))

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite, "session=%v", session)
		assert.True(t, cookies[0].HttpOnly)
		assert.True(t, cookies[0].Secure)
	}
}

func TestClearCookieTokenSameSite(t *testing.T) {
	config := &conf.Configuration{}
	config.Cookie.Key = "nf_jwt"
	ctx := withConfig(context.Background(), config)

	w := httptest.NewRecorder()
	api := &API{}
	api.clearCookieToken(ctx, w)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
	assert.True(t, cookies[0].HttpOnly)
	assert.True(t, cookies[0].Secure)
}
