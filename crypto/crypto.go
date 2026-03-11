package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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

// SecureCompare compares two strings in constant time without leaking
// their length. It hashes both inputs with SHA-256 first so that
// subtle.ConstantTimeCompare always operates on fixed-size (32-byte)
// digests regardless of input length.
func SecureCompare(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
