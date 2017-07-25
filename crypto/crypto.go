package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
)

// SecureToken creates a new random token
func SecureToken() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err.Error()) // rand should never fail
	}
	return removePadding(base64.URLEncoding.EncodeToString(b))
}

func removePadding(token string) string {
	return strings.TrimRight(token, "=")
}
