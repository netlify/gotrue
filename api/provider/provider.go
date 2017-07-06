package provider

import (
	"context"
	"golang.org/x/oauth2"
)

// Provider is an interface for interacting with external account providers
type Provider interface {
	GetUserEmail(context.Context, *oauth2.Token) (string, error)
	GetOAuthToken(context.Context, string) (*oauth2.Token, error)
}
