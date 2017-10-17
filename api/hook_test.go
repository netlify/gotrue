package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignupHook(t *testing.T) {
	var callCount int
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "", r.Header.Get("x-gotrue-signature"))

		defer squash(r.Body.Close)
		raw, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		data := map[string]string{}
		require.NoError(t, json.Unmarshal(raw, &data))
		assert.Equal(t, "myinstance", data["instance_id"])
		assert.Equal(t, "test@truth.com", data["email"])
		assert.Equal(t, "someone", data["provider"])

		w.WriteHeader(http.StatusOK)
	}))

	defer svr.Close()
	params := &SignupParams{
		Email:    "test@truth.com",
		Provider: "someone",
	}
	config := &conf.WebhookConfig{
		URL: svr.URL,
	}
	require.NoError(t, triggerSignupHook(params, "myinstance", "", config))

	assert.Equal(t, 1, callCount)
}

func TestSignupHookJWTSignature(t *testing.T) {
	var callCount int
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		signature := r.Header.Get("x-gotrue-signature")
		p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
		claims := new(jwt.StandardClaims)
		token, err := p.ParseWithClaims(signature, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("somesecret"), nil
		})
		require.NoError(t, err)

		assert.True(t, token.Valid)
		assert.Equal(t, "myinstance", claims.Subject)
		assert.WithinDuration(t, time.Now(), time.Unix(claims.IssuedAt, 0), 5*time.Second)
		w.WriteHeader(http.StatusOK)
	}))

	defer svr.Close()
	params := &SignupParams{
		Email:    "test@truth.com",
		Provider: "someone",
	}
	config := &conf.WebhookConfig{
		URL: svr.URL,
	}
	require.NoError(t, triggerSignupHook(params, "myinstance", "somesecret", config))

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
	require.NoError(t, w.trigger())

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
	err := w.trigger()
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
	err := w.trigger()
	require.Error(t, err)
	herr, ok := err.(*HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadGateway, herr.Code)
}

func squash(f func() error) { _ = f }
