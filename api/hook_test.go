package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignupHookSendInstanceID(t *testing.T) {
	user, err := models.NewUser("myinstance", "test@truth.com", "thisisapassword", "", nil)
	require.NoError(t, err)

	var callCount int
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		defer squash(r.Body.Close)
		raw, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		data := map[string]interface{}{}
		require.NoError(t, json.Unmarshal(raw, &data))

		assert.Len(t, data, 3)
		assert.Equal(t, "myinstance", data["instance_id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer svr.Close()

	config := &conf.Configuration{
		Webhook: conf.WebhookConfig{
			URL: svr.URL,
		},
	}

	require.NoError(t, triggerHook(SignupEvent, user, "myinstance", config))

	assert.Equal(t, 1, callCount)
}

func TestHookRetry(t *testing.T) {
	var callCount int
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		assert.EqualValues(t, 0, r.ContentLength)
		if callCount == 3 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer svr.Close()

	config := &conf.WebhookConfig{
		URL:     svr.URL,
		Retries: 3,
	}
	w := Webhook{
		WebhookConfig: config,
	}
	b, err := w.trigger()
	defer func() {
		if b != nil {
			b.Close()
		}
	}()
	require.NoError(t, err)

	assert.Equal(t, 3, callCount)
}

func TestHookTimeout(t *testing.T) {
	var callCount int
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		<-time.After(2 * time.Second)
	}))
	defer svr.Close()

	config := &conf.WebhookConfig{
		URL:        svr.URL,
		Retries:    3,
		TimeoutSec: 1,
	}
	w := Webhook{
		WebhookConfig: config,
	}
	_, err := w.trigger()
	require.Error(t, err)
	herr, ok := err.(*HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusGatewayTimeout, herr.Code)

	assert.Equal(t, 3, callCount)
}

func TestHookNoServer(t *testing.T) {
	config := &conf.WebhookConfig{
		URL:        "http://somewhere.something.com",
		Retries:    1,
		TimeoutSec: 1,
	}
	w := Webhook{
		WebhookConfig: config,
	}
	_, err := w.trigger()
	require.Error(t, err)
	herr, ok := err.(*HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadGateway, herr.Code)
}

func squash(f func() error) { _ = f }
