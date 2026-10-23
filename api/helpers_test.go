package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSMTPHost(t *testing.T) {
	for _, host := range []string{"", "8.8.8.8"} {
		assert.NoError(t, validateSMTPHost(host), host)
	}

	for _, host := range []string{
		"169.254.169.254", "127.0.0.1", "::1", "[::1]", "10.0.0.5",
		"localhost", "127.0.0.1/", "user@127.0.0.1", "8.8.8.8 ",
	} {
		assert.Error(t, validateSMTPHost(host), host)
	}
}
