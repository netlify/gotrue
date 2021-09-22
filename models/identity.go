package models

import (
	"database/sql"
	"time"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
)

type Identity struct {
	ID           string     `json:"id" db:"id"`
	UserID       uuid.UUID  `db:"user_id"`
	IdentityData JSONMap    `db:"identity_data"`
	Provider     string     `db:"provider"`
	LastSignInAt *time.Time `db:"last_sign_in_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

func (Identity) TableName() string {
	tableName := "identities"
	return tableName
}

// NewIdentity returns an identity associated to the user's id.
func NewIdentity(user *User, provider string, identityData map[string]interface{}) (*Identity, error) {
	id, ok := identityData["sub"]
	if !ok {
		return nil, errors.New("Error missing provider id")
	}
	now := time.Now()

	identity := &Identity{
		ID:           id.(string),
		UserID:       user.ID,
		IdentityData: identityData,
		Provider:     provider,
		LastSignInAt: &now,
	}

	return identity, nil
}

// FindIdentityById searches for an identity with the matching provider_id and provider given.
func FindIdentityByIdAndProvider(tx *storage.Connection, providerId, provider string) (*Identity, error) {
	identity := &Identity{}
	if err := tx.Q().Where("id = ? AND provider = ?", providerId, provider).First(identity); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, IdentityNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding identity")
	}
	return identity, nil
}

func FindIdentitiesByUser(tx *storage.Connection, user *User) ([]Identity, error) {
	var identities []Identity
	if err := tx.Q().Where("user_id = ?", user.ID).All(&identities); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return identities, nil
		}
		return nil, errors.Wrap(err, "error finding identities")
	}
	return identities, nil
}
