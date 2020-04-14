package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

const profileURL = "https://graph.facebook.com/me?fields=email,first_name,last_name,name,picture"

type facebookProvider struct {
	*oauth2.Config
}

type facebookUser struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Alias     string `json:"name"`
	Avatar    struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

// NewFacebookProvider creates a Facebook account provider.
func NewFacebookProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	return &facebookProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			RedirectURL:  ext.RedirectURI,
			Endpoint:     facebook.Endpoint,
			Scopes:       []string{"email"},
		},
	}, nil
}

func (p facebookProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	return p.Exchange(oauth2.NoContext, code)
}

func (p facebookProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	hash := hmac.New(sha256.New, []byte(p.Config.ClientSecret))
	hash.Write([]byte(tok.AccessToken))
	appsecretProof := hex.EncodeToString(hash.Sum(nil))

	var u facebookUser
	url := profileURL + "&appsecret_proof=" + appsecretProof
	if err := makeRequest(ctx, tok, p.Config, url, &u); err != nil {
		return nil, err
	}

	if u.Email == "" {
		return nil, errors.New("Unable to find email with Facebook provider")
	}

	return &UserProvidedData{
		Metadata: map[string]string{
			aliasKey:     u.Alias,
			nameKey:      strings.TrimSpace(u.FirstName + " " + u.LastName),
			avatarURLKey: u.Avatar.Data.URL,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: true,
			Primary:  true,
		}},
	}, nil
}
