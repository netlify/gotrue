package crypto

import (
	"encoding/base64"
	"strings"

	"github.com/pborman/uuid"
)

func SecureToken() string {
	token := uuid.NewRandom()
	return removePadding(base64.URLEncoding.EncodeToString([]byte(token)))
}

func removePadding(token string) string {
	return strings.TrimRight(token, "=")
}
