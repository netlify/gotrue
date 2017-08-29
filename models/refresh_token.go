package models

import (
	"time"
)

// RefreshToken is the database model for refresh tokens.
type RefreshToken struct {
	InstanceID string `json:"-"`
	ID         int64

	Token string

	User   User
	UserID string

	Revoked   bool
	CreatedAt time.Time
}

// TableName returns the database table name for RefreshToken
func (*RefreshToken) TableName() string {
	return tableName("refresh_tokens")
}
