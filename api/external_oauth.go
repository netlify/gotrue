package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/netlify/gotrue/api/provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// loadOAuthState parses the `state` query parameter as a JWS payload,
// extracting the provider requested
func (a *API) loadOAuthState(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return nil, badRequestError("OAuth state parameter missing")
	}

	ctx := r.Context()
	return a.loadExternalState(ctx, state)
}

func (a *API) oAuthCallback(ctx context.Context, r *http.Request, providerType string) (*provider.UserProvidedData, error) {
	rq := r.URL.Query()

	extError := rq.Get("error")
	if extError != "" {
		return nil, oauthError(extError, rq.Get("error_description"))
	}

	oauthCode := rq.Get("code")
	if oauthCode == "" {
		return nil, badRequestError("Authorization code missing")
	}

	oAuthProvider, err := a.OAuthProvider(ctx, providerType)
	if err != nil {
		return nil, badRequestError("Unsupported provider: %+v", err).WithInternalError(err)
	}

	log := getLogEntry(r)
	log.WithFields(logrus.Fields{
		"provider": providerType,
		"code":     oauthCode,
	}).Debug("Exchanging oauth code")

	tok, err := oAuthProvider.GetOAuthToken(oauthCode)
	if err != nil {
		return nil, internalServerError("Unable to exchange external code: %s", oauthCode).WithInternalError(err)
	}

	userData, err := oAuthProvider.GetUserData(ctx, tok)
	if err != nil {
		return nil, internalServerError("Error getting user email from external provider").WithInternalError(err)
	}

	config := a.getConfig(ctx)
	if config.External.TokenEncryptionKey != "" {
		cipher, err := encryptToken(config.External.TokenEncryptionKey, tok)
		if err != nil {
			log.WithError(err).Warn("Unable to encrypt oauth token for JWT payload")
		} else {
			if userData.AppMetadata == nil {
				userData.AppMetadata = make(map[string]interface{})
			}
			key := fmt.Sprintf("%s_token", providerType)
			userData.AppMetadata[key] = cipher
		}
	}

	return userData, nil
}

func encryptToken(pemKey string, token *oauth2.Token) (string, error) {
	// read pubkey
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil || block.Type != "PUBLIC KEY" {
		return "", errors.New("failed to decode PEM block containing public key")
	}

	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	pubKey, ok := parsedKey.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("Unable to parse RSA public key, generating a temp one")
	}

	secretMessage := []byte(token.AccessToken)
	label := []byte("token")

	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, secretMessage, label)
	if err != nil {
		return "", err
	}

	base64Cipher := base64.StdEncoding.EncodeToString(ciphertext)
	return base64Cipher, nil
}

func (a *API) OAuthProvider(ctx context.Context, name string) (provider.OAuthProvider, error) {
	providerCandidate, err := a.Provider(ctx, name)
	if err != nil {
		return nil, err
	}

	switch p := providerCandidate.(type) {
	case provider.OAuthProvider:
		return p, nil
	default:
		return nil, badRequestError("Provider can not be used for OAuth")
	}
}
