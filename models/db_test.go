package models_test

import (
	"testing"

	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/netlify/gotrue/storage/namespace"
)

func TestTableNameNamespacing(t *testing.T) {
	namespace.SetNamespace("test")
	assert.Equal(t, "test_audit_log_entries", models.AuditLogEntry{}.TableName())
	assert.Equal(t, "test_instances", models.Instance{}.TableName())
	assert.Equal(t, "test_refresh_tokens", models.RefreshToken{}.TableName())
	assert.Equal(t, "test_users", models.User{}.TableName())
}
