package models

import (
	"encoding/base64"
	"strings"

	"github.com/pborman/uuid"
)

func removePadding(token string) string {
	return strings.TrimRight(token, "=")
}

func secureToken() string {
	token := uuid.NewRandom()
	return removePadding(base64.URLEncoding.EncodeToString([]byte(token)))
}
