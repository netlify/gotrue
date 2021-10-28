package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const defaultSlackApiBase = "slack.com"

type slackProvider struct {
	*oauth2.Config
	APIPath string
}

type slackUser struct {
	ID        string `json:"https://slack.com/user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"picture"`
}

// NewSlackProvider creates a Slack account provider.
func NewSlackProvider(ext conf.OAuthProviderConfiguration, scopes string) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	apiPath := chooseHost(ext.URL, defaultSlackApiBase) + "/api"
	authPath := chooseHost(ext.URL, defaultSlackApiBase) + "/oauth"

	oauthScopes := []string{
		"profile",
		"email",
		"openid",
	}

	if scopes != "" {
		oauthScopes = append(oauthScopes, strings.Split(scopes, ",")...)
	}

	return &slackProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authPath + "/authorize",
				TokenURL: apiPath + "/oauth.access",
			},
			Scopes:      oauthScopes,
			RedirectURL: ext.RedirectURI,
		},
		APIPath: apiPath,
	}, nil
}

func (g slackProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(oauth2.NoContext, code)
}

func (g slackProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u slackUser
	if err := makeRequest(ctx, tok, g.Config, g.APIPath+"/openid.connect.userInfo", &u); err != nil {
		return nil, err
	}
	fmt.Printf("%+v\n", u)
	if u.Email == "" {
		return nil, errors.New("Unable to find email with Slack provider")
	}

	return &UserProvidedData{
		Metadata: &Claims{
			Issuer:        g.APIPath,
			Subject:       u.ID,
			Name:          u.Name,
			Picture:       u.AvatarURL,
			Email:         u.Email,
			EmailVerified: true, // Slack dosen't provide data on if email is verified.

			// To be deprecated
			AvatarURL:  u.AvatarURL,
			FullName:   u.Name,
			ProviderId: u.ID,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: true, // Slack dosen't provide data on if email is verified.
			Primary:  true,
		}},
	}, nil
}
