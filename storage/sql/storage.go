package sql

import (
	// this is where we do the connections

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/crypto"
	"github.com/netlify/netlify-auth/models"
	"github.com/pkg/errors"
)

type Connection struct {
	db     *gorm.DB
	config *conf.Configuration
}

func (conn *Connection) Close() error {
	return conn.db.Close()
}

func (conn *Connection) Automigrate(models ...interface{}) error {
	conn.db = conn.db.AutoMigrate(models...)
	return conn.db.Error
}

func (conn *Connection) CreateUser(user *models.User) error {
	obj := &UserObj{
		User:           user,
		FirstRoleName:  conn.config.JWT.AdminGroupName,
		AutoAsignRoles: conn.config.JWT.AdminGroupDisabled,
	}

	if result := conn.db.Create(obj); result.Error != nil {
		return errors.Wrap(result.Error, "Error creating user")
	}
	return nil
}

func (conn *Connection) FindUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	if result := conn.db.First(user, "email = ?", email); result.Error != nil {
		if result.RecordNotFound() {
			return nil, models.UserNotFoundError{}
		} else {
			return nil, errors.Wrap(result.Error, "error finding user")
		}
	}

	return user, nil
}

func (conn *Connection) FindUserByID(id string) (*models.User, error) {
	user := &models.User{}
	if result := conn.db.First(user, "id = ?", id); result.Error != nil {
		if result.RecordNotFound() {
			return nil, models.UserNotFoundError{}
		} else {
			return nil, errors.Wrap(result.Error, "error finding user")
		}
	}

	return user, nil
}

func (conn *Connection) FindUserByVerificationToken(verificationType models.VerifyType, token string) (*models.User, error) {
	user := &models.User{}
	if result := conn.db.First(user, "? = ?", verificationType, token); result.Error != nil {
		if result.RecordNotFound() {
			return nil, models.UserNotFoundError{}
		} else {
			return nil, errors.Wrap(result.Error, "error finding user")
		}
	}

	return user, nil
}

func (conn *Connection) FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	if result := conn.db.First(refreshToken, "token = ?", token); result.Error != nil {
		if result.RecordNotFound() {
			return nil, nil, models.RefreshTokenNotFoundError{}
		} else {
			return nil, nil, errors.Wrap(result.Error, "error finding refresh token")
		}
	}

	user := &models.User{}
	if result := conn.db.Model(refreshToken).Related(user); result.Error != nil {
		if result.RecordNotFound() {
			return nil, nil, models.UserNotFoundError{}
		} else {
			return nil, nil, errors.Wrap(result.Error, "error finding user")
		}
	}

	return user, refreshToken, nil
}

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

func (conn *Connection) IsDuplicatedEmail(email, id string) (bool, error) {
	user := &models.User{}
	result := conn.db.First(user, "id != ? and email = ?", id, email)
	if result.Error != nil {
		if result.RecordNotFound() {
			return false, nil
		}
		return false, errors.Wrap(result.Error, "error checking duplicated email")
	}

	return true, nil
}

func (conn *Connection) Logout(id interface{}) {
	conn.db.Where("user_id = ?", id).Delete(&models.RefreshToken{})
}

func (conn *Connection) RevokeToken(token *models.RefreshToken) error {
	token.Revoked = true
	if err := conn.db.Save(token).Error; err != nil {
		return errors.Wrap(err, "error revoking refresh token")
	}

	return nil
}

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

func (conn *Connection) UpdateUser(user *models.User) error {
	if result := conn.db.Model(user).Update(user); result.Error != nil {
		return errors.Wrap(result.Error, "Error updating user record")
	}
	return nil
}

// Connect will connect to that storage engine
func Connect(config *conf.Configuration) (*Connection, error) {
	db, err := gorm.Open(config.DB.Driver, config.DB.ConnURL)
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}

	err = db.DB().Ping()
	if err != nil {
		return nil, errors.Wrap(err, "checking database connection")
	}

	db.LogMode(true)

	conn := &Connection{
		db:     db,
		config: config,
	}

	return conn, nil
}

func createRefreshToken(tx *gorm.DB, user *models.User) (*models.RefreshToken, error) {
	token := &models.RefreshToken{
		User:  *user,
		Token: crypto.SecureToken(),
	}

	if err := tx.Create(token).Error; err != nil {
		return nil, errors.Wrap(err, "error creating refresh token")
	}

	return token, nil
}
