package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailedSignIn(t *testing.T) {
	u := &User{
		FailedSignInAttempts: 1,
	}
	u.FailedSignIn(1)
	assert.Equal(t, u.FailedSignInAttempts, 2)
	assert.NotNil(t, u.LockedAt)
	assert.True(t, u.IsLocked(10))
}

func TestIsLocked(t *testing.T) {
	timestamp := time.Now().Add(-time.Duration(5) * time.Minute)
	u := &User{
		LockedAt: &timestamp,
	}
	assert.False(t, u.IsLocked(0))
	assert.False(t, u.IsLocked(4))
	assert.False(t, u.IsLocked(5))
	assert.True(t, u.IsLocked(6))
}

func TestResetLock(t *testing.T) {
	now := time.Now()
	u := &User{
		LockedAt:             &now,
		FailedSignInAttempts: 3,
	}
	u.ResetLock()
	assert.Nil(t, u.LockedAt)
	assert.Equal(t, 0, u.FailedSignInAttempts)
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
