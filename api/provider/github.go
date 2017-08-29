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
func NewGithubProvider(ext conf.OAuthProviderConfiguration) Provider {
	return &githubProvider{
		&oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint:     github.Endpoint,
		},
	}
}

func (g githubProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g githubProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	u := struct {
		Email     string `json:"email"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}{}

	if err := makeRequest(ctx, tok, g.Config, "https://api.github.com/user", &u); err != nil {
		return nil, err
	}

	return &UserProvidedData{
		Email: u.Email,
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.AvatarURL,
		},
	}, nil
}
