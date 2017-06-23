package api

import (
	"context"
	"encoding/json"
	"errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type Provider interface {
	GetUserEmail(context.Context, *oauth2.Token) (string, error)
	GetOAuthToken(context.Context, string) (*oauth2.Token, error)
}

type githubProvider struct {
	*oauth2.Config
}

func NewGithubProvider(key, secret string) *githubProvider {
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

	userEmail := ""
	emails := []map[string]interface{}{}
	if err := json.NewDecoder(res.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, v := range emails {
		if primary, ok := v["primary"]; !ok {
			continue
		} else if p, ok := primary.(bool); !ok || !p {
			continue
		}

		if email, ok := v["email"]; ok {
			if e, ok := email.(string); ok {
				userEmail = e
			}
		}
	}

	if userEmail == "" {
		return "", errors.New("No email address returned by API call to github")
	}

	return userEmail, nil
}
