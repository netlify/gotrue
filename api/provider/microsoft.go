package provider

import (
	"context"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	microsoftGraphBaseUrl = "https://graph.microsoft.com/v1.0/"
)

type MicrosoftUser struct {
	Email string `json:"mail"`
	Name  string `json:"displayName"`
}

type microsoftProvider struct {
	*oauth2.Config
}

// NewMicrosoftProvider creates a Microsoft account provider.
func NewMicrosoftProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	return &microsoftProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			RedirectURL:  ext.RedirectURI,
			Endpoint:     microsoft.AzureADEndpoint(""),
			Scopes:       []string{"User.Read"},
		},
	}, nil
}

func (m microsoftProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return m.Exchange(oauth2.NoContext, code)
}

func (m microsoftProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {

	var u MicrosoftUser
	if err := makeRequest(ctx, tok, m.Config, microsoftGraphBaseUrl+"me", &u); err != nil {
		return nil, err
	}

	data := &UserProvidedData{
		Emails: []Email{{
			Email:    u.Email,
			Verified: true,
			Primary:  true,
		}},
		Metadata: map[string]string{
			nameKey: u.Name,
		},
	}

	return data, nil
}
