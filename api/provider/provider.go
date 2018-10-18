package provider

import (
	"golang.org/x/oauth2"
)

// Provider is an interface for interacting with external account providers
type Provider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
}

type UserProvidedData struct {
	Email    string
	Verified bool
	Metadata map[string]string
}
