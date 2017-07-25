package crypto

import (
	"encoding/base64"
	"strings"

	"github.com/pborman/uuid"
)

// SecureToken creates a new random token
func SecureToken() string {
	token := uuid.NewRandom()
	return removePadding(base64.URLEncoding.EncodeToString([]byte(token)))
}

func removePadding(token string) string {
	return strings.TrimRight(token, "=")
}
