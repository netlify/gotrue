package models_test

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"

	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
)

const modelsTestConfig = "../hack/test.env"

func TestTableNameNamespacing(t *testing.T) {
	cases := []struct {
		expected string
		value    interface{}
	}{
		{expected: "test_audit_log_entries", value: []*models.AuditLogEntry{}},
		{expected: "test_instances", value: []*models.Instance{}},
		{expected: "test_refresh_tokens", value: []*models.RefreshToken{}},
		{expected: "test_users", value: []*models.User{}},
	}

	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	conn, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	for _, tc := range cases {
		stmt := &gorm.Statement{DB: conn.DB}
		err := stmt.Parse(tc.value)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, stmt.Schema.Table)
	}
}
