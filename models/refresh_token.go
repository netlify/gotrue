package models

import (
	"time"

	"github.com/netlify/gotrue/storage/namespace"
	"github.com/netlify/gotrue/crypto"
	"github.com/pkg/errors"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"context"
	"github.com/tigrisdata/tigris-client-go/filter"
	"github.com/tigrisdata/tigris-client-go/fields"
	"github.com/google/uuid"
)

// RefreshToken is the database model for refresh tokens.
type RefreshToken struct {
	InstanceID uuid.UUID `json:"instance_id" db:"instance_id"`
	ID         uuid.UUID `json:"id" db:"id" tigris:"primaryKey"`

	Token string `json:"token" db:"token"`

	UserID uuid.UUID `json:"user_id" db:"user_id"`

	Revoked   bool      `json:"revoked" db:"revoked"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (RefreshToken) TableName() string {
	tableName := "refresh_tokens"

	if namespace.GetNamespace() != "" {
		return namespace.GetNamespace() + "_" + tableName
	}

	return tableName
}

// GrantAuthenticatedUser creates a refresh token for the provided user.
func GrantAuthenticatedUser(ctx context.Context, database *tigris.Database, user *User) (*RefreshToken, error) {
	return createRefreshToken(ctx, database, user)
}

// GrantRefreshTokenSwap swaps a refresh token for a new one, revoking the provided token.
func GrantRefreshTokenSwap(ctx context.Context, database *tigris.Database, user *User, token *RefreshToken) (*RefreshToken, error) {
	var newToken *RefreshToken
	err := database.Tx(ctx, func(ctx context.Context) error {
		var terr error
		if terr = NewAuditLogEntry(ctx, database, user.InstanceID, user, TokenRevokedAction, nil); terr != nil {
			return errors.Wrap(terr, "error creating audit log entry")
		}

		token.Revoked = true
		if _, terr = tigris.GetCollection[RefreshToken](database).Update(ctx, filter.Eq("id", token.ID), fields.Set("revoked", token.Revoked)); terr != nil {
			return terr
		}
		newToken, terr = createRefreshToken(ctx, database, user)
		return terr
	})
	return newToken, err
}

// Logout deletes all refresh tokens for a user.
func Logout(ctx context.Context, database *tigris.Database, instanceID uuid.UUID, id uuid.UUID) error {
	_, err := tigris.GetCollection[RefreshToken](database).Delete(ctx, filter.And(filter.Eq("instance_id", instanceID), filter.Eq("user_id", id)))
	return err
}

func createRefreshToken(ctx context.Context, database *tigris.Database, user *User) (*RefreshToken, error) {
	token := &RefreshToken{
		InstanceID: user.InstanceID,
		UserID:     user.ID,
		Token:      crypto.SecureToken(),
		ID:         uuid.New(),
	}

	if _, err := tigris.GetCollection[RefreshToken](database).Insert(ctx, token); err != nil {
		return nil, errors.Wrap(err, "error creating refresh token")
	}
	return token, nil
}
