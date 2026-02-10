package storage

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

type TestUser struct {
	ID    uuid.UUID
	Role  string `db:"role"`
	Other string `db:"othercol"`
}

func TestGetExcludedColumns(t *testing.T) {
	u := TestUser{}
	cols, err := getExcludedColumns(u, "role")
	require.NoError(t, err)
	require.NotContains(t, cols, "role")
	require.Contains(t, cols, "othercol")
}

func TestGetExcludedColumns_InvalidName(t *testing.T) {
	u := TestUser{}
	_, err := getExcludedColumns(u, "adsf")
	require.Error(t, err)
}
