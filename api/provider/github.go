package provider

import (
	"context"
	"encoding/json"
	"errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Github

type githubProvider struct {
	*oauth2.Config
}

func NewGithubProvider(key, secret string) Provider {
	return &githubProvider{
		&oauth2.Config{
			ClientID:     key,
			ClientSecret: secret,
			Endpoint:     github.Endpoint,
		},
	}
}

func (g githubProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g githubProvider) GetUserEmail(ctx context.Context, tok *oauth2.Token) (string, error) {
	client := g.Client(ctx, tok)
	res, err := client.Get("https://api.github.com/user/emails")
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

	return "", errors.New("No email address returned by API call to github")
}
