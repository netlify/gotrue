package api

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/netlify/gotrue/conf"
	"github.com/pkg/errors"
)

// JWKS - public REST endpoints
type JWKS struct {
	handler      http.Handler
	globalConfig *conf.GlobalConfiguration
	config       *conf.Configuration
	version      string
	publicKey    rsa.PublicKey
	response     map[string]interface{}
}

// NewJKWS - constructs newer JWKS endpoint
func NewJKWS(globalConfig *conf.GlobalConfiguration, config *conf.Configuration, version string) (*JWKS, error) {
	result := &JWKS{
		handler:      nil,
		globalConfig: globalConfig,
		config:       config,
		version:      version,
	}

	var keysMap []map[string]interface{}

	for _, key := range config.JWT.RSAPublicKeys {
		publicKeyPEM, err := os.ReadFile(key)
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(publicKeyPEM)
		if block == nil || block.Type != "PUBLIC KEY" {
			return nil, errors.New("Couldn't decode pem file")
		}

		publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, errors.New("Couldn't parse pkix public key")
		}

		rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not of type RSA")
		}

		//  thumbprint
		publicKeyDER, err := x509.MarshalPKIXPublicKey(rsaPublicKey)
		if err != nil {
			return nil, err
		}
		thumbprint := sha1.Sum(publicKeyDER)
		hexThumbprint := hex.EncodeToString(thumbprint[:])

		// kid
		kid, err := getKeyID(rsaPublicKey)
		if err != nil {
			return nil, err
		}

		thisKeyInfo := map[string]interface{}{"kty": "RSA",
			"alg": config.JWT.Algorithm,
			"use": "sig",
			"kid": kid,
			"n":   base64UrlEncode(rsaPublicKey.N.Bytes()),
			"e":   base64UrlEncode(big.NewInt(int64(rsaPublicKey.E)).Bytes()),
			"x5t": hexThumbprint}

		keysMap = append(keysMap, thisKeyInfo)
	}

	result.response = map[string]interface{}{
		"keys": keysMap,
	}
	return result, nil
}

// getJWKS returns a public key information
func (a *JWKS) getJWKS(w http.ResponseWriter, _ *http.Request) error {
	return sendJSON(w, http.StatusOK, a.response)
}

func base64UrlEncode(input []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(input), "=")
}

func getKeyID(pubKey *rsa.PublicKey) (string, error) {
	h := crypto.SHA256.New()
	if _, err := h.Write(pubKey.N.Bytes()); err != nil {
		return "", err
	}
	kid := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return kid, nil
}
