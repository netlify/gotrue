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

type gitlabUser struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url"`
	ConfirmedAt string `json:"confirmed_at"`
}

type gitlabUserEmail struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

// NewGitlabProvider creates a Gitlab account provider.
func NewGitlabProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

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
	}, nil
}

func (g gitlabProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(oauth2.NoContext, code)
}

func (g gitlabProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u gitlabUser

	if err := makeRequest(ctx, tok, g.Config, g.Host+"/api/v4/user", &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.AvatarURL,
		},
	}

	var emails []*gitlabUserEmail
	if err := makeRequest(ctx, tok, g.Config, g.Host+"/api/v4/user/emails", &emails); err != nil {
		return nil, err
	}

	for _, e := range emails {
		// additional emails from GitLab don't return confirm status
		if e.Email != "" {
			data.Emails = append(data.Emails, Email{Email: e.Email, Verified: false, Primary: false})
		}
	}

	if u.Email != "" {
		verified := u.ConfirmedAt != ""
		data.Emails = append(data.Emails, Email{Email: u.Email, Verified: verified, Primary: true})
	}

	if len(data.Emails) <= 0 {
		return nil, errors.New("Unable to find email with GitLab provider")
	}

	return data, nil
}
