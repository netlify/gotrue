package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const (
	defaultGoogleAuthBase = "accounts.google.com"
	defaultGoogleAPIBase  = "www.googleapis.com"
)

type googleProvider struct {
	*oauth2.Config
	APIPath string
}

type googleUser struct {
	Name          string `json:"name"`
	AvatarURL     string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"verified_email"`
}

// NewGoogleProvider creates a Google account provider.
func NewGoogleProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	authHost := chooseHost(ext.URL, defaultGoogleAuthBase)
	apiPath := chooseHost(ext.URL, defaultGoogleAPIBase) + "/userinfo/v2/me"

	return &googleProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authHost + "/o/oauth2/auth",
				TokenURL: authHost + "/o/oauth2/token",
			},
			Scopes: []string{
				"email",
				"profile",
			},
			RedirectURL: ext.RedirectURI,
		},
		APIPath: apiPath,
	}, nil
}

func (g googleProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(oauth2.NoContext, code)
}

func (g googleProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u googleUser
	if err := makeRequest(ctx, tok, g.Config, g.APIPath, &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.AvatarURL,
		},
	}

	if u.Email != "" {
		data.Emails = append(data.Emails, Email{
			Email:    u.Email,
			Verified: u.EmailVerified,
			Primary:  true,
		})
	}

	if len(data.Emails) <= 0 {
		return nil, errors.New("Unable to find email with Google provider")
	}

	return data, nil
}
