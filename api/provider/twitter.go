package provider

import (
	"context"
	"fmt"
	"net/http"

	"errors"

	"github.com/mrjones/oauth"
	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

var (
	requestURL      = "https://api.twitter.com/oauth/request_token"
	authorizeURL    = "https://api.twitter.com/oauth/authorize"
	authenticateURL = "https://api.twitter.com/oauth/authenticate"
	tokenURL        = "https://api.twitter.com/oauth/access_token"
	endpointProfile = "https://api.twitter.com/1.1/account/verify_credentials.json"
)

type twitterProvider struct {
	ClientKey    string
	Secret       string
	CallbackURL  string
	HTTPClient   *http.Client
	debug        bool
	consumer     *oauth.Consumer
	providerName string
}

type twitterUser struct {
	Name          string `json:"name"`
	AvatarURL     string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// NewTwitterProvider creates a Twitter account provider.
func NewTwitterProvider(ext conf.OAuthProviderConfiguration) (Provider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	t := &twitterProvider{
		ClientKey:    ext.ClientID,
		Secret:       ext.Secret,
		CallbackURL:  ext.RedirectURI,
		providerName: "twitter",
	}
	t.consumer = newConsumer(t, authorizeURL)

	return t, nil
}

func (t twitterProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	requestToken, url, err := t.consumer.GetRequestTokenAndUrl(t.CallbackURL)

	if err != nil {
		return nil, err
	}

	tok := &oauth2.Token{
		AccessToken:  requestToken.Token, // TODO Dont be bad, this is different type
		TokenType:    url,
		RefreshToken: requestToken.Secret,
	}
	return tok, nil
}

func (t twitterProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var accessToken := tok.AccessToken
	var u twitterUser

	if accessToken == "" {
		// data is not yet retrieved since accessToken is still empty
		return nil, fmt.Errorf("%s cannot get user information without accessToken", t.providerName)
	}

	response, err := t.consumer.Get(
		endpointProfile,
		map[string]string{"include_entities": "false", "skip_status": "true", "include_email": "true"},
		*accessToken)
}

func (t twitterProvider) AuthCodeURL(string, ...oauth2.AuthCodeOption) (string, error) {
	return "nil", errors.New("Refresh token is not provided by twitter")
}

func newConsumer(provider *twitterProvider, authURL string) *oauth.Consumer {
	c := oauth.NewConsumer(
		provider.ClientKey,
		provider.Secret,
		oauth.ServiceProvider{
			RequestTokenUrl:   requestURL,
			AuthorizeTokenUrl: authURL,
			AccessTokenUrl:    tokenURL,
		})

	c.Debug(provider.debug)
	return c
}
