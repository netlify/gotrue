package models

import (
	"time"

	"github.com/netlify/gotrue/storage/namespace"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
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

func (RefreshToken) TableName() string {
	tableName := "refresh_tokens"

	if namespace.GetNamespace() != "" {
		return namespace.GetNamespace() + "_" + tableName
	}

	return tableName
}

// GrantAuthenticatedUser creates a refresh token for the provided user.
func GrantAuthenticatedUser(tx *storage.Connection, user *User) (*RefreshToken, error) {
	return createRefreshToken(tx, user)
}

// GrantRefreshTokenSwap swaps a refresh token for a new one, revoking the provided token.
func GrantRefreshTokenSwap(tx *storage.Connection, user *User, token *RefreshToken) (*RefreshToken, error) {
	var newToken *RefreshToken
	err := tx.Transaction(func(rtx *storage.Connection) error {
		var terr error
		if terr = NewAuditLogEntry(tx, user.InstanceID, user, TokenRevokedAction, nil); terr != nil {
			return errors.Wrap(terr, "error creating audit log entry")
		}

		token.Revoked = true
		if terr = tx.UpdateOnly(token, "revoked"); terr != nil {
			return terr
		}
		newToken, terr = createRefreshToken(rtx, user)
		return terr
	})
	return newToken, err
}

// Logout deletes all refresh tokens for a user.
func Logout(tx *storage.Connection, instanceID uuid.UUID, id uuid.UUID) error {
	return tx.RawQuery("DELETE FROM "+(&pop.Model{Value: RefreshToken{}}).TableName()+" WHERE instance_id = ? AND user_id = ?", instanceID, id).Exec()
}

func createRefreshToken(tx *storage.Connection, user *User) (*RefreshToken, error) {
	token := &RefreshToken{
		InstanceID: user.InstanceID,
		UserID:     user.ID,
		Token:      crypto.SecureToken(),
	}

	if err := tx.Create(token); err != nil {
		return nil, errors.Wrap(err, "error creating refresh token")
	}
	return token, nil
}
