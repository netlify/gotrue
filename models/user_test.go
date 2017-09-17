package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsLocked(t *testing.T) {
	timestamp := time.Now().Add(-time.Duration(5) * time.Minute)
	u := &User{
		LockedAt: &timestamp,
	}
	require.False(t, u.IsLocked(0), "user should not be locked")
	require.False(t, u.IsLocked(4), "user should not be locked")
	require.False(t, u.IsLocked(5), "user should not be locked")
	require.True(t, u.IsLocked(6), "user should be locked")
}

func TestResetLock(t *testing.T) {
	now := time.Now()
	u := &User{
		LockedAt:             &now,
		FailedSignInAttempts: 3,
	}
	u.ResetLock()
	require.Nil(t, u.LockedAt, "LockedAt should not be set")
	require.Equal(t, 0, u.FailedSignInAttempts, "FailedSignInAttempts should be 0")
}

func TestUpdateAppMetadata(t *testing.T) {
	u := &User{}
	u.UpdateAppMetaData(make(map[string]interface{}))

	require.NotNil(t, u.AppMetaData)

	u.UpdateAppMetaData(map[string]interface{}{
		"foo": "bar",
	})

	require.Equal(t, "bar", u.AppMetaData["foo"])
	u.UpdateAppMetaData(map[string]interface{}{
		"foo": nil,
	})
	require.Len(t, u.AppMetaData, 0)
	require.Equal(t, nil, u.AppMetaData["foo"])
}

func TestUpdateUserMetadata(t *testing.T) {
	u := &User{}
	u.UpdateUserMetaData(make(map[string]interface{}))

	require.NotNil(t, u.UserMetaData)

	u.UpdateUserMetaData(map[string]interface{}{
		"foo": "bar",
	})

	require.Equal(t, "bar", u.UserMetaData["foo"])
	u.UpdateUserMetaData(map[string]interface{}{
		"foo": nil,
	})
	require.Len(t, u.UserMetaData, 0)
	require.Equal(t, nil, u.UserMetaData["foo"])
}
