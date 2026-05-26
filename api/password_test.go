package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"empty is accepted here", "", false},
		{"short password", "hunter2", false},
		{"exactly 72 bytes", strings.Repeat("a", 72), false},
		{"73 bytes is rejected", strings.Repeat("a", 73), true},
		// 4-byte emoji * 19 = 76 bytes total
		{"long emoji string rejected", strings.Repeat("😀", 19), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.wantErr {
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
