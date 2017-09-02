package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

// Gitlab

const defaultGitLabAuthBase = "gitlab.com"

type gitlabProvider struct {
	*oauth2.Config
	Host string
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

// NewGitlabProvider creates a Gitlab account provider.
func NewGitlabProvider(ext conf.OAuthProviderConfiguration) Provider {
	host := chooseHost(ext.URL, defaultGitLabAuthBase)
	return &gitlabProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  host + "/oauth/authorize",
				TokenURL: host + "/oauth/token",
			},
			RedirectURL: ext.RedirectURI,
			Scopes:      []string{"read_user"},
		},
		Host: host,
	}
}

func (g gitlabProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g gitlabProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u githubUser

	if err := makeRequest(ctx, tok, g.Config, g.Host+"/api/v4/user", &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.AvatarURL,
		},
	}

	var emails []*githubUserEmail
	if err := makeRequest(ctx, tok, g.Config, g.Host+"/api/v4/user/emails", &emails); err != nil {
		return nil, err
	}

	if len(emails) > 0 {
		data.Email = emails[0].Email
	}

	if data.Email == "" {
		return nil, errors.New("Unable to find email with GitLab provider")
	}

	return data, nil
}
