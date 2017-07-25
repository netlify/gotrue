package provider

import (
	"context"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Github

type githubProvider struct {
	*oauth2.Config
}

// NewGithubProvider creates a Github account provider.
func NewGithubProvider(ext conf.ExternalConfiguration) Provider {
	return &githubProvider{
		&oauth2.Config{
			ClientID:     ext.Key,
			ClientSecret: ext.Secret,
			Endpoint:     github.Endpoint,
		},
	}
}

func (g githubProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g githubProvider) GetUserEmail(ctx context.Context, tok *oauth2.Token) (string, error) {
	return getUserEmail(ctx, tok, g.Config, "https://api.github.com/user/emails")
}
