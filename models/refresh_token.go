package models

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// RefreshToken is the database model for refresh tokens.
type RefreshToken struct {
	InstanceID uuid.UUID `json:"-" db:"instance_id"`
	ID         int64     `db:"id"`

	Token string `db:"token"`

	UserID uuid.UUID `db:"user_id"`

	Revoked   bool      `db:"revoked"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
