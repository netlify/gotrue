package provider

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

// Gitlab

type gitlabProvider struct {
	*oauth2.Config
}

var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://gitlab.com/api/v4/oauth/authorize",
	TokenURL: "https://gitlab.com/api/v4/oauth/token",
}

func NewGitlabProvider(key, secret string) Provider {
	return &gitlabProvider{
		&oauth2.Config{
			ClientID:     key,
			ClientSecret: secret,
			Endpoint:     Endpoint,
		},
	}
}

func (g gitlabProvider) GetOAuthToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.Exchange(ctx, code)
}

func (g gitlabProvider) GetUserEmail(ctx context.Context, tok *oauth2.Token) (string, error) {
	client := g.Client(ctx, tok)
	res, err := client.Get("https://gitlab.com/api/v4/user")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	user := struct {
		Email string `json:"email"`
	}{}

	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return "", err
	}

	return user.Email, nil
}
