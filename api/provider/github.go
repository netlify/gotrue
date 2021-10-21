package provider

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

// Github

const (
	defaultGitHubAuthBase = "github.com"
	defaultGitHubAPIBase  = "api.github.com"
)

type githubProvider struct {
	*oauth2.Config
	APIHost string
}

type githubUser struct {
	ID        int    `json:"id"`
	UserName  string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type githubUserEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// NewGithubProvider creates a Github account provider.
func NewGithubProvider(ext conf.OAuthProviderConfiguration, scopes string) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	authHost := chooseHost(ext.URL, defaultGitHubAuthBase)
	apiHost := chooseHost(ext.URL, defaultGitHubAPIBase)
	if !strings.HasSuffix(apiHost, defaultGitHubAPIBase) {
		apiHost += "/api/v3"
	}

	oauthScopes := []string{
		"user:email",
	}

	if scopes != "" {
		oauthScopes = append(oauthScopes, strings.Split(scopes, ",")...)
	}

	return &githubProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authHost + "/login/oauth/authorize",
				TokenURL: authHost + "/login/oauth/access_token",
			},
			RedirectURL: ext.RedirectURI,
			Scopes:      oauthScopes,
		},
		APIHost: apiHost,
	}, nil
}

func (g githubProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(oauth2.NoContext, code)
}

func (g githubProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u githubUser
	if err := makeRequest(ctx, tok, g.Config, g.APIHost+"/user", &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Metadata: &Claims{
			Issuer:            g.APIHost,
			Subject:           strconv.Itoa(u.ID),
			Name:              u.Name,
			PreferredUsername: u.UserName,

			// To be deprecated
			AvatarURL:   u.AvatarURL,
			FullName:    u.Name,
			ProviderId:  strconv.Itoa(u.ID),
			UserNameKey: u.UserName,
		},
	}

	var emails []*githubUserEmail
	if err := makeRequest(ctx, tok, g.Config, g.APIHost+"/user/emails", &emails); err != nil {
		return nil, err
	}

	for _, e := range emails {
		if e.Email != "" {
			data.Emails = append(data.Emails, Email{Email: e.Email, Verified: e.Verified, Primary: e.Primary})
		}

		if e.Primary {
			data.Metadata.Email = e.Email
			data.Metadata.EmailVerified = e.Verified
		}
	}

	if len(data.Emails) <= 0 {
		return nil, errors.New("Unable to find email with GitHub provider")
	}

	return data, nil
}
