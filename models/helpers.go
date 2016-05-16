package models

import "crypto/rand"

func secureToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Reader.Read(bytes)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
