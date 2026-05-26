package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordRejectsOverlongInput(t *testing.T) {
	_, err := hashPassword(strings.Repeat("a", MaxPasswordLength+1))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooLong)
}

func TestHashPasswordAcceptsBoundary(t *testing.T) {
	_, err := hashPassword(strings.Repeat("a", MaxPasswordLength))
	require.NoError(t, err)
}
