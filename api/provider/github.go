package provider

import (
	"context"

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
	return getUserEmail(ctx, tok, "https://api.github.com/user/emails", g.Config)
}
