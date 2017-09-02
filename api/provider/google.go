package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleBaseURL = "https://www.googleapis.com/oauth2/v2/userinfo"

type googleProvider struct {
	*oauth2.Config
}

type googleUser struct {
	Name          string `json:"name"`
	AvatarURL     string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// NewGoogleProvider creates a Google account provider.
func NewGoogleProvider(ext conf.OAuthProviderConfiguration) (Provider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	return &googleProvider{
		&oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
			},
			RedirectURL: ext.RedirectURI,
		},
	}, nil
}

func (g googleProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(oauth2.NoContext, code)
}

func (g googleProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u googleUser
	if err := makeRequest(ctx, tok, g.Config, googleBaseURL, &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Email:    u.Email,
		Verified: u.EmailVerified,
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.AvatarURL,
		},
	}

	if data.Email == "" {
		return nil, errors.New("Unable to find email with Google provider")
	}

	return data, nil
}
