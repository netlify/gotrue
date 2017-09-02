package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleBaseURL = "https://www.googleapis.com/plus/v1/people/me"

type googleProvider struct {
	*oauth2.Config
}

type googleUser struct {
	Name   string `json:"displayName"`
	Avatar struct {
		URL string `json:"url"`
	} `json:"image"`
	Emails []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"emails"`
}

// NewGoogleProvider creates a Google account provider.
func NewGoogleProvider(ext conf.OAuthProviderConfiguration) Provider {
	return &googleProvider{
		&oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.profile",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			RedirectURL: ext.RedirectURI,
		},
	}
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
		Verified: true,
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.Avatar.URL,
		},
	}

	if len(u.Emails) > 0 {
		data.Email = u.Emails[0].Value
	}

	if data.Email == "" {
		return nil, errors.New("Unable to find email with Google provider")
	}

	return data, nil
}
