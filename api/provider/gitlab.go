package provider

import (
	"context"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

// Gitlab

type gitlabProvider struct {
	*oauth2.Config
	External conf.OAuthProviderConfiguration
}

func defaultBase(base string) string {
	if base == "" {
		return "https://gitlab.com"
	}

	baseLen := len(base)
	if base[baseLen-1] == '/' {
		return base[:baseLen-1]
	}

	return base
}

// NewGitlabProvider creates a Gitlab account provider.
func NewGitlabProvider(ext conf.OAuthProviderConfiguration) Provider {
	base := defaultBase(ext.URL)
	return &gitlabProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  base + "/oauth/authorize",
				TokenURL: base + "/oauth/token",
			},
			RedirectURL: ext.RedirectURI,
		},
		External: ext,
	}
}

func (g gitlabProvider) VerifiesEmails() bool {
	return false
}

func (g gitlabProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g gitlabProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	user := struct {
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}{}
	base := defaultBase(g.External.URL)

	if err := makeRequest(ctx, tok, g.Config, base+"/api/v4/user", &user); err != nil {
		return nil, err
	}

	return &UserProvidedData{
		Email: user.Email,
		Metadata: map[string]string{
			nameKey:      user.Name,
			avatarURLKey: user.AvatarURL,
		},
	}, nil
}
