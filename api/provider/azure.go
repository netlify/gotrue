package provider

import (
	"context"
	"errors"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const (
	defaultAzureAuthBase = "login.microsoftonline.com"
	defaultAzureAPIBase  = "graph.microsoft.com"
)

type azureProvider struct {
	*oauth2.Config
	APIPath string
}

type azureUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type azureEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"is_primary"`
	Verified bool   `json:"is_confirmed"`
}

// NewAzureProvider creates a Azure account provider.
func NewAzureProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	authHost := chooseHost(ext.URL, defaultAzureAuthBase)
	apiPath := chooseHost(ext.URL, defaultAzureAPIBase)

	return &azureProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authHost + "/common/oauth2/v2.0/authorize",
				TokenURL: authHost + "/common/oauth2/v2.0/token",
			},
			RedirectURL: ext.RedirectURI,
			Scopes:      []string{"openid"},
		},
		APIPath: apiPath,
	}, nil
}

func (g azureProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return g.Exchange(oauth2.NoContext, code)
}

func (g azureProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var u azureUser
	if err := makeRequest(ctx, tok, g.Config, g.APIPath+"/oidc/userinfo", &u); err != nil {
		return nil, err
	}

	if u.Email == "" {
		return nil, errors.New("Unable to find email with Azure provider")
	}

	return &UserProvidedData{
		Metadata: map[string]string{
			nameKey: u.Name,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: true,
			Primary:  true,
		}},
	}, nil
}
