package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

// Bitbucket

const (
	bitbucketBaseURL   = "https://api.bitbucket.org/2.0/user"
	bitbucketEmailsURL = bitbucketBaseURL + "/emails"
)

type bitbucketProvider struct {
	*oauth2.Config
}

type bitbucketUser struct {
	Name   string `json:"display_name"`
	Avatar struct {
		Href string `json:"href"`
	} `json:"avatar"`
}

type bitbucketEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"is_primary"`
	Verified bool   `json:"is_confirmed"`
}

type bitbucketEmails struct {
	Values []bitbucketEmail `json:"values"`
}

// NewBitbucketProvider creates a Bitbucket account provider.
func NewBitbucketProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	return &bitbucketProvider{
		&oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint:     bitbucket.Endpoint,
			RedirectURL:  ext.RedirectURI,
			Scopes:       []string{"account", "email"},
		},
	}, nil
}

func (g bitbucketProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(oauth2.NoContext, code)
}

func (g bitbucketProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u bitbucketUser
	if err := makeRequest(ctx, tok, g.Config, bitbucketBaseURL, &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Metadata: map[string]string{
			nameKey:      u.Name,
			avatarURLKey: u.Avatar.Href,
		},
	}

	var emails bitbucketEmails
	if err := makeRequest(ctx, tok, g.Config, bitbucketEmailsURL, &emails); err != nil {
		return nil, err
	}

	if len(emails.Values) > 0 {
		for _, e := range emails.Values {
			data.Emails = append(data.Emails, Email{
				Email:    e.Email,
				Verified: e.Verified,
				Primary:  e.Primary,
			})
		}
	}

	if len(data.Emails) <= 0 {
		return nil, errors.New("Unable to find email with Bitbucket provider")
	}

	return data, nil
}
