package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mrjones/oauth"
	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const (
	requestURL      = "https://api.twitter.com/oauth/request_token"
	authenticateURL = "https://api.twitter.com/oauth/authenticate"
	tokenURL        = "https://api.twitter.com/oauth/access_token"
	endpointProfile = "https://api.twitter.com/1.1/account/verify_credentials.json"
)

type TwitterProvider struct {
	ClientKey     string
	Secret        string
	CallbackURL   string
	AuthURL       string
	RequestToken  *oauth.RequestToken
	OauthVerifier string
	Consumer      *oauth.Consumer
}

type twitterUser struct {
	UserName  string `json:"screen_name"`
	Name      string `json:"name"`
	AvatarURL string `json:"profile_image_url"`
	Email     string `json:"email"`
}

func NewTwitterProvider(ext conf.OAuthProviderConfiguration, scopes string) (OAuthProvider, error) {
	p := &TwitterProvider{
		ClientKey:   ext.ClientID,
		Secret:      ext.Secret,
		CallbackURL: ext.RedirectURI,
	}
	p.Consumer = newConsumer(p)
	return p, nil
}

func (t TwitterProvider) GetOAuthToken(_ string) (*oauth2.Token, error) {
	// stub method for OAuthProvider interface, unused in OAuth1.0 protocol
	return &oauth2.Token{}, nil
}

func (t TwitterProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	// stub method for OAuthProvider interface, unused in OAuth1.0 protocol
	return &UserProvidedData{}, nil
}

func (t TwitterProvider) FetchUserData(ctx context.Context, tok *oauth.AccessToken) (*UserProvidedData, error) {
	var u twitterUser
	resp, err := t.Consumer.Get(
		endpointProfile,
		map[string]string{"include_entities": "false", "skip_status": "true", "include_email": "true"},
		tok)
	if err != nil {
		return &UserProvidedData{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &UserProvidedData{}, fmt.Errorf("Twitter responded with a %d trying to fetch user information", resp.StatusCode)
	}
	bits, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &UserProvidedData{}, nil
	}
	err = json.NewDecoder(bytes.NewReader(bits)).Decode(&u)

	data := &UserProvidedData{
		Metadata: map[string]string{
			userNameKey:  u.UserName,
			nameKey:      u.Name,
			avatarURLKey: u.AvatarURL,
		},
		Emails: []Email{{
			Email:    u.Email,
			Verified: true,
			Primary:  true,
		}},
	}
	return data, nil
}

func (t *TwitterProvider) AuthCodeURL(state string, args ...oauth2.AuthCodeOption) string {
	// we do nothing with the state here as the state is passed in the requestURL step
	requestToken, url, err := t.Consumer.GetRequestTokenAndUrl(t.CallbackURL + "?state=" + state)
	if err != nil {
		return ""
	}
	t.RequestToken = requestToken
	t.AuthURL = url
	return t.AuthURL
}

func newConsumer(provider *TwitterProvider) *oauth.Consumer {
	c := oauth.NewConsumer(
		provider.ClientKey,
		provider.Secret,
		oauth.ServiceProvider{
			RequestTokenUrl:   requestURL,
			AuthorizeTokenUrl: authenticateURL,
			AccessTokenUrl:    tokenURL,
		})
	return c
}

func (t TwitterProvider) Marshal() string {
	b, _ := json.Marshal(t.RequestToken)
	return string(b)
}

func (t TwitterProvider) Unmarshal(data string) (*oauth.RequestToken, error) {
	requestToken := &oauth.RequestToken{}
	err := json.NewDecoder(strings.NewReader(data)).Decode(requestToken)
	return requestToken, err
}
