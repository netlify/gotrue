package provider

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

const (
	avatarURLKey = "avatar_url"
	nameKey      = "full_name"
	aliasKey     = "slug"
)

type UserProvidedData struct {
	Email    string
	Verified bool
	Metadata map[string]string
}

// Provider is an interface for interacting with external account providers
type Provider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	GetUserData(context.Context, *oauth2.Token) (*UserProvidedData, error)
	GetOAuthToken(string) (*oauth2.Token, error)
}

func makeRequest(ctx context.Context, tok *oauth2.Token, g *oauth2.Config, url string, dst interface{}) error {
	client := g.Client(ctx, tok)
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		return err
	}

	return nil
}
