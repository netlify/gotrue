package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

// Bitbucket

type bitbucketProvider struct {
	*oauth2.Config
}

// NewBitbucketProvider creates a Bitbucket account provider.
func NewBitbucketProvider(ext conf.ExternalConfiguration) Provider {
	return &bitbucketProvider{
		&oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint:     bitbucket.Endpoint,
		},
	}
}

func (g bitbucketProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g bitbucketProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	u := struct {
		User struct {
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			AvatarURL string `json:"avatar"`
		}
	}{}

	if err := makeRequest(ctx, tok, g.Config, "https://api.bitbucket.org/1.0/user", &u); err != nil {
		return nil, err
	}

	email, err := getUserEmail(ctx, tok, g.Config, fmt.Sprintf("https://api.bitbucket.org/1.0/users/%s/emails", u.User.Username))
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(strings.Join([]string{u.User.FirstName, u.User.LastName}, " "))

	return &UserProvidedData{
		Email: email,
		Metadata: map[string]string{
			nameKey:      name,
			avatarURLKey: u.User.AvatarURL,
		},
	}, nil
}

func getUserEmail(ctx context.Context, tok *oauth2.Token, g *oauth2.Config, url string) (string, error) {
	emails := []struct {
		Primary bool   `json:"primary"`
		Email   string `json:"email"`
	}{}

	if err := makeRequest(ctx, tok, g, url, &emails); err != nil {
		return "", err
	}

	for _, v := range emails {
		if !v.Primary {
			continue
		}
		return v.Email, nil
	}

	return "", errors.New("No email address returned by API call to " + url)
}
