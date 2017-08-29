package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleProvider struct {
	*oauth2.Config
}

// NewGoogleProvider creates a Google account provider.
func NewGoogleProvider(ext conf.OAuthProviderConfiguration) Provider {
	return &googleProvider{
		&oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"profile", "email"},
		},
	}
}

func (g googleProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g googleProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	u := struct {
		Name   string `json:"displayName"`
		Avatar struct {
			URL string `json:"url"`
		} `json:"image"`
		Emails []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"emails"`
	}{}

	if err := makeRequest(ctx, tok, g.Config, "https://www.googleapis.com/plus/v1/people/me", &u); err != nil {
		return nil, err
	}

	var email string
	if len(u.Emails) > 0 {
		email = u.Emails[0].Value
	}
	if email == "" {
		return nil, errors.New("No email address returned by Google")
	}

	return &UserProvidedData{
		Email: email,
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.Avatar.URL,
		},
	}, nil
}
