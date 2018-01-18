package models

import (
	"time"

	"github.com/markbates/pop"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
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

func revokeToken(tx *storage.Connection, token *RefreshToken, revoked bool) error {
	token.Revoked = revoked
	return tx.UpdateOnly(token, "revoked")
}

// RevokeToken revokes a refresh token.
func RevokeToken(tx *storage.Connection, token *RefreshToken) error {
	return revokeToken(tx, token, true)
}

// RollbackRefreshTokenSwap rolls back a refresh token swap by revoking the new
// token, and un-revoking the old token.
func RollbackRefreshTokenSwap(tx *storage.Connection, newToken, oldToken *RefreshToken) error {
	return tx.Transaction(func(rtx *storage.Connection) error {
		if err := revokeToken(rtx, newToken, true); err != nil {
			return err
		}
		return revokeToken(rtx, oldToken, false)
	})
}

// GrantAuthenticatedUser creates a refresh token for the provided user.
func GrantAuthenticatedUser(tx *storage.Connection, user *User) (*RefreshToken, error) {
	return createRefreshToken(tx, user)
}

// GrantRefreshTokenSwap swaps a refresh token for a new one, revoking the provided token.
func GrantRefreshTokenSwap(tx *storage.Connection, user *User, token *RefreshToken) (*RefreshToken, error) {
	var newToken *RefreshToken
	err := tx.Transaction(func(rtx *storage.Connection) error {
		terr := revokeToken(rtx, token, true)
		if terr != nil {
			return terr
		}
		newToken, terr = createRefreshToken(rtx, user)
		return terr
	})
	return newToken, err
}

// Logout deletes all refresh tokens for a user.
func Logout(tx *storage.Connection, id uuid.UUID) {
	tx.RawQuery("DELETE FROM "+(&pop.Model{Value: RefreshToken{}}).TableName()+" WHERE user_id = ?", id).Exec()
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
