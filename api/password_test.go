package api

import (
	"strings"
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePassword(t *testing.T) {
	strict := &conf.Configuration{Security: conf.SecurityConfiguration{
		Strict:            true,
		MinPasswordLength: 8,
	}}
	legacy := &conf.Configuration{}

	cases := []struct {
		name     string
		config   *conf.Configuration
		password string
		wantErr  bool
	}{
		{"legacy empty", legacy, "", false},
		{"legacy single char", legacy, "a", false},
		{"legacy at 72 bytes", legacy, strings.Repeat("a", 72), false},
		{"legacy 73 bytes rejected", legacy, strings.Repeat("a", 73), true},
		// 4-byte emoji * 19 = 76 bytes total
		{"legacy long emoji rejected", legacy, strings.Repeat("😀", 19), true},
		{"strict empty (callers handle)", strict, "", false},
		{"strict below min", strict, "short", true},
		{"strict at min", strict, "12345678", false},
		{"strict above min", strict, "longer-password", false},
		{"strict 73 bytes rejected (max takes precedence)", strict, strings.Repeat("a", 73), true},
		// Rune-count semantics: 3 emoji = 12 bytes but only 3 characters,
		// so it must fail an 8-char policy. 8 emoji = 32 bytes / 8 chars,
		// so it must pass.
		{"strict 3 emoji rejected as 3 chars", strict, strings.Repeat("😀", 3), true},
		{"strict 8 emoji accepted as 8 chars", strict, strings.Repeat("😀", 8), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.config, tc.password)
			if tc.wantErr {
				require.Error(t, err)
				he, ok := err.(*HTTPError)
				require.True(t, ok, "expected *HTTPError, got %T", err)
				assert.Equal(t, 422, he.Code)
				return
			}
			require.NoError(t, err)
		})
	}
}
