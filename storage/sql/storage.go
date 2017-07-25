package sql

import (
	// this is where we do the connections
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/sirupsen/logrus"
	"github.com/jinzhu/gorm"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/models"
	"github.com/pkg/errors"
)

type logger struct {
	entry *logrus.Entry
}

func (l logger) Print(v ...interface{}) {
	l.entry.Print(v...)
}

// Connection represents a sql connection.
type Connection struct {
	db     *gorm.DB
	config *conf.Configuration
}

// Automigrate creates any missing tables and/or columns.
func (conn *Connection) Automigrate() error {
	conn.db = conn.db.AutoMigrate(&userObj{}, &models.RefreshToken{})
	return conn.db.Error
}

// Close closes the database connection.
func (conn *Connection) Close() error {
	return conn.db.Close()
}

// CreateUser creates a user.
func (conn *Connection) CreateUser(user *models.User) error {
	tx := conn.db.Begin()
	if _, err := conn.createUserWithTransaction(tx, user); err != nil {
		return err
	}
	tx.Commit()
	return nil
}

func (conn *Connection) createUserWithTransaction(tx *gorm.DB, user *models.User) (*userObj, error) {
	obj := conn.newUserObj(user)
	if result := tx.Create(obj); result.Error != nil {
		tx.Rollback()
		return nil, errors.Wrap(result.Error, "Error creating user")
	}

	return obj, nil
}

func (conn *Connection) findUser(query string, args ...interface{}) (*models.User, error) {
	obj := &userObj{
		User: &models.User{},
	}
	values := append([]interface{}{query}, args...)

	if result := conn.db.First(obj, values...); result.Error != nil {
		if result.RecordNotFound() {
			return nil, models.UserNotFoundError{}
		}
		return nil, errors.Wrap(result.Error, "error finding user")
	}

	return obj.User, nil
}

// DeleteUser deletes a user.
func (conn *Connection) DeleteUser(u *models.User) error {
	return conn.db.Delete(u).Error
}

// FindUsersInAudience finds users with the matching audience.
func (conn *Connection) FindUsersInAudience(aud string) ([]*models.User, error) {
	users := []*models.User{}
	db := conn.db.Find(&users, "aud = ?", aud)
	return users, db.Error
}

// FindUserByConfirmationToken finds users with the matching confirmation token.
func (conn *Connection) FindUserByConfirmationToken(token string) (*models.User, error) {
	return conn.findUser("confirmation_token = ?", token)
}

// FindUserByEmailAndAudience finds a user with the matching email and audience.
func (conn *Connection) FindUserByEmailAndAudience(email, aud string) (*models.User, error) {
	return conn.findUser("email = ? and aud = ?", email, aud)
}

// FindUserByID finds a user matching the provided ID.
func (conn *Connection) FindUserByID(id string) (*models.User, error) {
	return conn.findUser("id = ?", id)
}

// FindUserByRecoveryToken finds a user with the matching recovery token.
func (conn *Connection) FindUserByRecoveryToken(token string) (*models.User, error) {
	return conn.findUser("recovery_token = ?", token)
}

// FindUserWithRefreshToken finds a user from the provided refresh token.
func (conn *Connection) FindUserWithRefreshToken(token, aud string) (*models.User, *models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	if result := conn.db.First(refreshToken, "token = ?", token); result.Error != nil {
		if result.RecordNotFound() {
			return nil, nil, models.RefreshTokenNotFoundError{}
		}
		return nil, nil, errors.Wrap(result.Error, "error finding refresh token")
	}

	user, err := conn.findUser("id = ? and aud = ?", refreshToken.UserID, aud)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// GrantAuthenticatedUser creates a refresh token for the provided user.
func (conn *Connection) GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error) {
	tx := conn.db.Begin()

	tx.Save(user)

	token, err := createRefreshToken(tx, user)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return token, nil
}

// GrantRefreshTokenSwap swaps a refresh token for a new one, revoking the provided token.
func (conn *Connection) GrantRefreshTokenSwap(user *models.User, token *models.RefreshToken) (*models.RefreshToken, error) {
	tx := conn.db.Begin()

	token.Revoked = true
	if err := tx.Save(token).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	newToken, err := createRefreshToken(tx, user)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return newToken, nil
}

// IsDuplicatedEmail returns whether a user exists with a matching email and audience.
func (conn *Connection) IsDuplicatedEmail(email, aud string) (bool, error) {
	_, err := conn.findUser("email = ? and aud = ?", email, aud)
	if err != nil {
		if models.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Logout deletes all refresh tokens for a user.
func (conn *Connection) Logout(id string) {
	conn.db.Where("user_id = ?", id).Delete(&models.RefreshToken{})
}

// RevokeToken revokes a refresh token.
func (conn *Connection) RevokeToken(token *models.RefreshToken) error {
	token.Revoked = true
	if err := conn.db.Save(token).Error; err != nil {
		return errors.Wrap(err, "error revoking refresh token")
	}

	return nil
}

// RollbackRefreshTokenSwap rolls back a refresh token swap by revoking the new
// token, and un-revoking the old token.
func (conn *Connection) RollbackRefreshTokenSwap(newToken, oldToken *models.RefreshToken) error {
	tx := conn.db.Begin()

	newToken.Revoked = true
	if err := tx.Save(newToken).Error; err != nil {
		tx.Rollback()
		return err
	}

	oldToken.Revoked = false
	if err := tx.Save(oldToken).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// UpdateUser updates a user.
func (conn *Connection) UpdateUser(user *models.User) error {
	tx := conn.db.Begin()
	if err := conn.updateUserWithTransaction(tx, user); err != nil {
		return err
	}
	tx.Commit()
	return nil
}

func (conn *Connection) updateUserWithTransaction(tx *gorm.DB, user *models.User) error {
	obj := &userObj{
		User: user,
	}
	if result := tx.Save(obj); result.Error != nil {
		tx.Rollback()
		return errors.Wrap(result.Error, "Error updating user record")
	}
	return nil
}

// Dial will connect to that storage engine
func Dial(config *conf.Configuration) (*Connection, error) {
	db, err := gorm.Open(config.DB.Driver, config.DB.ConnURL)
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}

	if err := db.DB().Ping(); err != nil {
		return nil, errors.Wrap(err, "checking database connection")
	}

	db.SetLogger(logger{logrus.WithField("db-connection", config.DB.Driver)})

	if config.Logging.IsDebugEnabled() {
		db.LogMode(true)
	}

	conn := &Connection{
		db:     db,
		config: config,
	}

	return conn, nil
}

func createRefreshToken(tx *gorm.DB, user *models.User) (*models.RefreshToken, error) {
	token := &models.RefreshToken{
		User:   *user,
		UserID: user.ID,
		Token:  crypto.SecureToken(),
	}

	if err := tx.Create(token).Error; err != nil {
		return nil, errors.Wrap(err, "error creating refresh token")
	}

	return token, nil
}
