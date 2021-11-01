package provider

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

type Claims struct {
	// Reserved claims
	Issuer  string  `json:"iss,omitempty"`
	Subject string  `json:"sub,omitempty"`
	Aud     string  `json:"aud,omitempty"`
	Iat     float64 `json:"iat,omitempty"`
	Exp     float64 `json:"exp,omitempty"`

	// Default profile claims
	Name              string `json:"name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	MiddleName        string `json:"middle_name,omitempty"`
	NickName          string `json:"nickname,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Profile           string `json:"profile,omitempty"`
	Picture           string `json:"picture,omitempty"`
	Website           string `json:"website,omitempty"`
	Gender            string `json:"gender,omitempty"`
	Birthdate         string `json:"birthdate,omitempty"`
	ZoneInfo          string `json:"zoneinfo,omitempty"`
	Locale            string `json:"locale,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
	Phone             string `json:"phone,omitempty"`
	PhoneVerified     bool   `json:"phone_verified,omitempty"`

	// Custom profile claims that are provider specific
	CustomClaims map[string]interface{} `json:"custom_claims,omitempty"`

	// TODO: Deprecate in next major release
	FullName    string `json:"full_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Slug        string `json:"slug,omitempty"`
	ProviderId  string `json:"provider_id,omitempty"`
	UserNameKey string `json:"user_name,omitempty"`
}

// ToMap converts the Claims struct to a map[string]interface{}
func (c *Claims) ToMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	cBytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(cBytes, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Email is a struct that provides information on whether an email is verified or is the primary email address
type Email struct {
	Email    string
	Verified bool
	Primary  bool
}

// UserProvidedData is a struct that contains the user's data returned from the oauth provider
type UserProvidedData struct {
	Emails   []Email
	Metadata *Claims
}

// Provider is an interface for interacting with external account providers
type Provider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
}

// OAuthProvider specifies additional methods needed for providers using OAuth
type OAuthProvider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	GetUserData(context.Context, *oauth2.Token) (*UserProvidedData, error)
	GetOAuthToken(string) (*oauth2.Token, error)
}

func chooseHost(base, defaultHost string) string {
	if base == "" {
		return "https://" + defaultHost
	}

	baseLen := len(base)
	if base[baseLen-1] == '/' {
		return base[:baseLen-1]
	}

	return base
}

func makeRequest(ctx context.Context, tok *oauth2.Token, g *oauth2.Config, url string, dst interface{}) error {
	client := g.Client(ctx, tok)
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		return err
	}

	return nil
}
