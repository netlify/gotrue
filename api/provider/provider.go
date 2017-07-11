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

func getUserEmail(ctx context.Context, tok *oauth2.Token, g *oauth2.Config, url string) (string, error) {
	emails := []struct {
		Primary bool   `json:"primary"`
		Email   string `json:"email"`
	}{}

	if err := makeRequest(ctx, tok, g, url, &emails); err != nil {
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
