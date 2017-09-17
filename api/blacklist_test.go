package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBlacklistUpdate(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Duration(25) * time.Hour)
	bl := Blacklist{}

	require.True(t, bl.UpdateNeeded(), "blacklist should require an update")
	bl.Update(make(map[string]bool))
	require.False(t, bl.UpdateNeeded(), "blacklist should not require an update")

	bl.updatedAt = &earlier
	require.True(t, bl.UpdateNeeded(), "blacklist should require update after 24 hours")
}

func TestBlacklistEmail(t *testing.T) {
	bl := Blacklist{}
	domains := make(map[string]bool)
	domains["mailinator.com"] = true
	bl.Update(domains)

	require.True(t, bl.EmailBlacklisted("test@mailinator.com"), "mailinator.com should be blacklisted")
	require.True(t, bl.EmailBlacklisted("test2@mailinator.com"), "mailinator.com should be blacklisted")
	require.False(t, bl.EmailBlacklisted("test@example.com"), "example.com should not be blacklisted")
}
