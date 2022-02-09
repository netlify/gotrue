package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2"
)

const (
	avatarURLKey = "avatar_url"
	nameKey      = "full_name"
	aliasKey     = "slug"
)

type Email struct {
	Email    string
	Verified bool
	Primary  bool
}

type UserProvidedData struct {
	Emails   []Email
	Metadata map[string]string
}

// Provider is an interface for interacting with external account providers
type Provider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
}

// OAuthProvider specifies additional methods needed for providers using OAuth
type OAuthProvider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	GetUserData(context.Context, *oauth2.Token) (*UserProvidedData, error)
	GetOAuthToken(string) (*oauth2.Token, error)
}

func chooseHost(base, defaultHost string) string {
	if base == "" {
		return "https://" + defaultHost
	}

	baseLen := len(base)
	if base[baseLen-1] == '/' {
		return base[:baseLen-1]
	}

	return base
}

type RequestError struct {
	code int
	body string
}

func (r *RequestError) Error() string {
	return fmt.Sprintf("Request failed with status %d:\n%s", r.code, r.body)
}

func makeRequest(ctx context.Context, tok *oauth2.Token, g *oauth2.Config, url string, dst interface{}) error {
	client := g.Client(ctx, tok)
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return &RequestError{code: res.StatusCode, body: string(body)}
	}

	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		return err
	}

	return nil
}
