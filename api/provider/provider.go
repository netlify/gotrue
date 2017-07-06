package provider

import (
	"context"
	"encoding/json"
	"errors"

	"golang.org/x/oauth2"
)

// Provider is an interface for interacting with external account providers
type Provider interface {
	GetUserEmail(context.Context, *oauth2.Token) (string, error)
	GetOAuthToken(context.Context, string) (*oauth2.Token, error)
}

func getUserEmail(ctx context.Context, tok *oauth2.Token, url string, g *oauth2.Config) (string, error) {
	client := g.Client(ctx, tok)
	res, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	emails := []struct {
		Primary bool   `json:"primary"`
		Email   string `json:"email"`
	}{}

	if err := json.NewDecoder(res.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, v := range emails {
		if !v.Primary {
			continue
		}
		return v.Email, nil
	}

	return "", errors.New("No email address returned by API call to " + url)
}
