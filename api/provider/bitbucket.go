package provider

import (
	"context"
	"fmt"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

// Bitbucket

type bitbucketProvider struct {
	*oauth2.Config
}

// NewBitbucketProvider creates a Bitbucket account provider.
func NewBitbucketProvider(ext conf.ExternalConfiguration) Provider {
	return &bitbucketProvider{
		&oauth2.Config{
			ClientID:     ext.Key,
			ClientSecret: ext.Secret,
			Endpoint:     bitbucket.Endpoint,
		},
	}
}

func (g bitbucketProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g bitbucketProvider) GetUserEmail(ctx context.Context, tok *oauth2.Token) (string, error) {
	u := struct {
		User struct {
			Username string `json:"username"`
		}
	}{}

	if err := makeRequest(ctx, tok, g.Config, "https://api.bitbucket.org/1.0/user", &u); err != nil {
		return "", err
	}

	return getUserEmail(ctx, tok, g.Config, fmt.Sprintf("https://api.bitbucket.org/1.0/users/%s/emails", u.User.Username))
}
