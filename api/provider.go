package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
	"golang.org/x/oauth2/github"
)

// Provider is an interface for interacting with external account providers
type Provider interface {
	GetUserEmail(context.Context, *oauth2.Token) (string, error)
	GetOAuthToken(context.Context, string) (*oauth2.Token, error)
}

var knownProviders = map[string]func(key, secret string) Provider{
	"github":    NewGithubProvider,
	"bitbucket": NewBitbucketProvider,
}

// Provider returns a Provider inerface for the given name
func (a *API) Provider(name string) (Provider, error) {
	fn, ok := knownProviders[name]
	if !ok {
		return nil, errors.New("Unknown provider")
	}

	name = strings.ToLower(name)

	var key, secret string
	switch name {
	case "github":
		key, secret = a.config.External.Github.Key, a.config.External.Github.Secret
	case "bitbucket":
		key, secret = a.config.External.Bitbucket.Key, a.config.External.Bitbucket.Secret
	}

	if key == "" || secret == "" {
		return nil, fmt.Errorf("Provider %s could not be found", name)
	}

	return fn(key, secret), nil
}

// Github

type githubProvider struct {
	*oauth2.Config
}

func NewGithubProvider(key, secret string) Provider {
	return &githubProvider{
		&oauth2.Config{
			ClientID:     key,
			ClientSecret: secret,
			Endpoint:     github.Endpoint,
		},
	}
}

func (g githubProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g githubProvider) GetUserEmail(ctx context.Context, tok *oauth2.Token) (string, error) {
	client := g.Client(ctx, tok)
	res, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err

	}
	defer res.Body.Close()

	userEmail := ""
	emails := []map[string]interface{}{}
	if err := json.NewDecoder(res.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, v := range emails {
		if primary, ok := v["primary"]; !ok {
			continue
		} else if p, ok := primary.(bool); !ok || !p {
			continue
		}

		if email, ok := v["email"]; ok {
			if e, ok := email.(string); ok {
				userEmail = e
			}
		}
	}

	if userEmail == "" {
		return "", errors.New("No email address returned by API call to github")
	}

	return userEmail, nil
}

// Bitbucket

type bitbucketProvider struct {
	*oauth2.Config
}

func NewBitbucketProvider(key, secret string) Provider {
	return &bitbucketProvider{
		&oauth2.Config{
			ClientID:     key,
			ClientSecret: secret,
			Endpoint:     bitbucket.Endpoint,
		},
	}
}

func (g bitbucketProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

type bitbucketProviderUser struct {
	User map[string]interface{} `json:"user"`
}

func (g bitbucketProvider) GetUserEmail(ctx context.Context, tok *oauth2.Token) (string, error) {
	client := g.Client(ctx, tok)
	userRes, err := client.Get("https://api.bitbucket.org/1.0/user")
	if err != nil {
		return "", err
	}
	defer userRes.Body.Close()

	username := ""
	u := map[string]interface{}{}
	if err := json.NewDecoder(userRes.Body).Decode(&u); err != nil {
		return "", err
	}

	x, ok := u["user"]
	u, ok2 := x.(map[string]interface{})
	if !ok || !ok2 {
		return "", errors.New("Invalid response when requesting email address from bitbucket")

	}

	if name, ok := u["username"]; ok {
		username, ok = name.(string)
		if !ok {
			return "", errors.New("Invalid response when requesting email address from bitbucket")
		}
	}

	res, err := client.Get(fmt.Sprintf("https://api.bitbucket.org/1.0/users/%s/emails", username))
	if err != nil {
		return "", err

	}
	defer res.Body.Close()

	userEmail := ""
	emails := []map[string]interface{}{}
	if err := json.NewDecoder(res.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, v := range emails {
		if primary, ok := v["primary"]; !ok {
			continue
		} else if p, ok := primary.(bool); !ok || !p {
			continue
		}

		if email, ok := v["email"]; ok {
			if e, ok := email.(string); ok {
				userEmail = e
			}
		}
	}

	if userEmail == "" {
		return "", errors.New("No email address returned by API call to bitbucket")
	}

	return userEmail, nil
}
