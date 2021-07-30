package provider

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/netlify/gotrue/conf"
	"golang.org/x/oauth2"
)

const (
	authEndpoint  = "https://appleid.apple.com/auth/authorize"
	tokenEndpoint = "https://appleid.apple.com/auth/token"

	ScopeEmail = "email"
	ScopeName  = "name"

	appleAudOrIss                  = "https://appleid.apple.com"
	idTokenVerificationKeyEndpoint = "https://appleid.apple.com/auth/keys"
)

type AppleProvider struct {
	*oauth2.Config
	APIPath    string
	httpClient *http.Client
}

type appleName struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type appleUser struct {
	Name  appleName `json:"name"`
	Email string    `json:"email"`
}

type idTokenClaims struct {
	jwt.StandardClaims
	AccessTokenHash string `json:"at_hash"`
	AuthTime        int    `json:"auth_time"`
	Email           string `json:"email"`
	IsPrivateEmail  bool   `json:"is_private_email,string"`
}

func NewAppleProvider(ext conf.OAuthProviderConfiguration) (OAuthProvider, error) {
	if err := ext.Validate(); err != nil {
		return nil, err
	}

	return &AppleProvider{
		Config: &oauth2.Config{
			ClientID:     ext.ClientID,
			ClientSecret: ext.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authEndpoint,
				TokenURL: tokenEndpoint,
			},
			Scopes: []string{
				ScopeEmail,
				ScopeName,
			},
			RedirectURL: ext.RedirectURI,
		},
		APIPath: "",
	}, nil
}

func (p AppleProvider) GetOAuthToken(code string) (*oauth2.Token, error) {
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("client_id", p.ClientID),
		oauth2.SetAuthURLParam("secret", p.ClientSecret),
	}
	return p.Exchange(oauth2.NoContext, code, opts...)
}

func (p AppleProvider) GetUserData(ctx context.Context, tok *oauth2.Token) (*UserProvidedData, error) {
	var user *UserProvidedData
	if tok.AccessToken == "" {
		return &UserProvidedData{}, nil
	}
	if idToken := tok.Extra("id_token"); idToken != nil {
		idToken, err := jwt.ParseWithClaims(idToken.(string), &idTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
			kid := t.Header["kid"].(string)
			claims := t.Claims.(*idTokenClaims)
			vErr := new(jwt.ValidationError)
			if !claims.VerifyAudience(p.ClientID, true) {
				vErr.Inner = fmt.Errorf("incorrect audience")
				vErr.Errors |= jwt.ValidationErrorAudience
			}
			if !claims.VerifyIssuer(appleAudOrIss, true) {
				vErr.Inner = fmt.Errorf("incorrect issuer")
				vErr.Errors |= jwt.ValidationErrorIssuer
			}
			if vErr.Errors > 0 {
				return nil, vErr
			}

			// per OpenID Connect Core 1.0 §3.2.2.9, Access Token Validation
			hash := sha256.Sum256([]byte(tok.AccessToken))
			halfHash := hash[0:(len(hash) / 2)]
			encodedHalfHash := base64.RawURLEncoding.EncodeToString(halfHash)
			if encodedHalfHash != claims.AccessTokenHash {
				vErr.Inner = fmt.Errorf(`invalid identity token`)
				vErr.Errors |= jwt.ValidationErrorClaimsInvalid
				return nil, vErr
			}

			// get the public key for verifying the identity token signature
			set, err := jwk.FetchHTTP(idTokenVerificationKeyEndpoint, jwk.WithHTTPClient(http.DefaultClient))
			if err != nil {
				return nil, err
			}
			selectedKey := set.Keys[0]
			for _, key := range set.Keys {
				if key.KeyID() == kid {
					selectedKey = key
					break
				}
			}
			pubKeyIface, _ := selectedKey.Materialize()
			pubKey, ok := pubKeyIface.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf(`expected RSA public key from %s`, idTokenVerificationKeyEndpoint)
			}
			return pubKey, nil
		})
		if err != nil {
			return &UserProvidedData{}, err
		}
		user = &UserProvidedData{
			Emails: []Email{{
				Email:    idToken.Claims.(*idTokenClaims).Email,
				Verified: true,
				Primary:  true,
			}},
		}

	}
	return user, nil
}

func (p AppleProvider) ParseUser(data string) map[string]string {
	userData := &appleUser{}
	err := json.Unmarshal([]byte(data), userData)
	if err != nil {
		return nil
	}
	return map[string]string{
		"firstName": userData.Name.FirstName,
		"lastName":  userData.Name.LastName,
		"email":     userData.Email,
	}
}
