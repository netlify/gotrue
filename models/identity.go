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
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	IdentityData JSONMap    `json:"identity_data,omitempty" db:"identity_data"`
	Provider     string     `json:"provider" db:"provider"`
	LastSignInAt *time.Time `json:"last_sign_in_at,omitempty" db:"last_sign_in_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
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

// FindIdentitiesByUser returns all identities associated to a user
func FindIdentitiesByUser(tx *storage.Connection, user *User) ([]*Identity, error) {
	identities := []*Identity{}
	if err := tx.Q().Where("user_id = ?", user.ID).All(&identities); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return identities, nil
		}
		return nil, errors.Wrap(err, "error finding identities")
	}
	return identities, nil
}

// FindProvidersByUser returns all providers associated to a user
func FindProvidersByUser(tx *storage.Connection, user *User) ([]string, error) {
	identities := []Identity{}
	providers := make([]string, 0)
	if err := tx.Q().Select("provider").Where("user_id = ?", user.ID).All(&identities); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return providers, nil
		}
		return nil, errors.Wrap(err, "error finding providers")
	}
	for _, identity := range identities {
		providers = append(providers, identity.Provider)
	}
	return providers, nil
}
