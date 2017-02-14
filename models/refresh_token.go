package models

import "time"

type RefreshToken struct {
	ID int64

	Token string

	User   User
	UserID string

	Revoked   bool
	CreatedAt time.Time
}

func (RefreshToken) TableName() string {
	return tableName("refresh_tokens")
}
