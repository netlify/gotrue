package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
