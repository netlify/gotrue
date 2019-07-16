package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

var netlifyOauthEndpoint = oauth2.Endpoint{
	AuthURL:  "https://app.netlify.com/authorize",
	TokenURL: "https://api.netlify.com/oauth/token",
}

const netlifyUserEndpoint = "https://api.netlify.com/api/v1/user"

type netlifyProvider struct {
	*oauth2.Config
}

type netlifyUser struct {
	AvatarURL string `json:"avatar_url,omitempty"`
	Email     string `json:"email,omitempty"`
	FullName  string `json:"full_name,omitempty"`
	ID        string `json:"id,omitempty"`
}

// NewNetlifyProvider provides a configured provider for logging in with Netlify
func NewNetlifyProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	return &netlifyProvider{
		&oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint:     netlifyOauthEndpoint,
			RedirectURL:  ext.RedirectURI,
		},
	}, nil
}

func (n netlifyProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return n.Exchange(oauth2.NoContext, code)
}

func (n netlifyProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u netlifyUser
	if err := makeRequest(ctx, tok, n.Config, netlifyUserEndpoint, &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Email:    u.Email,
		Verified: true,
		Metadata: map[string]string{
			nameKey:      u.FullName,
			avatarURLKey: u.AvatarURL,
		},
	}

	if data.Email == "" {
		return nil, errors.New("Unable to find email with Netlify provider")
	}

	return data, nil
}
