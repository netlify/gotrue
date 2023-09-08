package provider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/netlify/gotrue/api/provider"
	"github.com/netlify/gotrue/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestGithubFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
		fmt.Fprint(rw, "Something failed")
	}))
	t.Cleanup(srv.Close)

	gh, err := provider.NewGithubProvider(conf.OAuthProviderConfiguration{
		ClientID:    "client-id",
		Secret:      "secret",
		RedirectURI: "https://redirect.example.org/callback",
		URL:         srv.URL,
		Enabled:     true,
	})
	require.NoError(t, err)

	user, err := gh.GetUserData(ctx, &oauth2.Token{
		AccessToken: "my-token",
		Expiry:      time.Now().Add(time.Minute),
	})
	require.Error(t, err)
	require.Nil(t, user)
	assert.Equal(t, "Request failed with status 418:\nSomething failed", err.Error())
}
