package sql

import (
	// this is where we do the connections

	"fmt"
	"net/url"

	// import drivers we might need
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/jinzhu/gorm"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type logger struct {
	entry *logrus.Entry
}

func (l logger) Print(v ...interface{}) {
	l.entry.Print(v...)
}

// Connection represents a sql connection.
type Connection struct {
	db *gorm.DB
}

// Automigrate creates any missing tables and/or columns.
func (conn *Connection) Automigrate() error {
	conn.db = conn.db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.Instance{})
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

// CountOtherUsers counts how many other users exist besides the one provided
func (conn *Connection) CountOtherUsers(instanceID string, id string) (int, error) {
	u := models.User{}
	var userCount int
	if result := conn.db.Table(u.TableName()).Where("instance_id = ? and id != ?", instanceID, id).Count(&userCount); result.Error != nil {
		return 0, errors.Wrap(result.Error, "error finding registered users")
	}
	return userCount, nil
}

func (conn *Connection) createUserWithTransaction(tx *gorm.DB, user *models.User) (*models.User, error) {
	if result := tx.Create(user); result.Error != nil {
		tx.Rollback()
		return nil, errors.Wrap(result.Error, "Error creating user")
	}

	return user, nil
}

func (conn *Connection) findUser(query string, args ...interface{}) (*models.User, error) {
	obj := &models.User{}
	values := append([]interface{}{query}, args...)

	if result := conn.db.First(obj, values...); result.Error != nil {
		if result.RecordNotFound() {
			return nil, models.UserNotFoundError{}
		}
		return nil, errors.Wrap(result.Error, "error finding user")
	}

	return obj, nil
}

// DeleteUser deletes a user.
func (conn *Connection) DeleteUser(u *models.User) error {
	return conn.db.Delete(u).Error
}

// FindUsersInAudience finds users with the matching audience.
func (conn *Connection) FindUsersInAudience(instanceID string, aud string, pageParams *models.Pagination, sortParams *models.SortParams) ([]*models.User, error) {
	users := []*models.User{}
	q := conn.db.Table((&models.User{}).TableName()).Where("instance_id = ? and aud = ?", instanceID, aud)

	if sortParams != nil && len(sortParams.Fields) > 0 {
		for _, field := range sortParams.Fields {
			q = q.Order(field.Name + " " + string(field.Dir))
		}
	}

	var rsp *gorm.DB
	if pageParams != nil {
		var total uint64
		if cq := q.Count(&total); cq.Error != nil {
			return nil, cq.Error
		}
		pageParams.Count = total

		rsp = q.Offset(pageParams.Offset()).Limit(pageParams.PerPage).Find(&users)
	} else {
		rsp = q.Find(&users)
	}

	return users, rsp.Error
}

// FindUserByConfirmationToken finds users with the matching confirmation token.
func (conn *Connection) FindUserByConfirmationToken(token string) (*models.User, error) {
	return conn.findUser("confirmation_token = ?", token)
}

// FindUserByEmailAndAudience finds a user with the matching email and audience.
func (conn *Connection) FindUserByEmailAndAudience(instanceID, email, aud string) (*models.User, error) {
	return conn.findUser("instance_id = ? and email = ? and aud = ?", instanceID, email, aud)
}

// FindUserByID finds a user matching the provided ID.
func (conn *Connection) FindUserByID(id string) (*models.User, error) {
	return conn.findUser("id = ?", id)
}

// FindUserByInstanceIDAndID finds a user matching the provided ID.
func (conn *Connection) FindUserByInstanceIDAndID(instanceID, id string) (*models.User, error) {
	return conn.findUser("instance_id = ? and id = ?", instanceID, id)
}

// FindUserByRecoveryToken finds a user with the matching recovery token.
func (conn *Connection) FindUserByRecoveryToken(token string) (*models.User, error) {
	return conn.findUser("recovery_token = ?", token)
}

// FindUserWithRefreshToken finds a user from the provided refresh token.
func (conn *Connection) FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	if result := conn.db.First(refreshToken, "token = ?", token); result.Error != nil {
		if result.RecordNotFound() {
			return nil, nil, models.RefreshTokenNotFoundError{}
		}
		return nil, nil, errors.Wrap(result.Error, "error finding refresh token")
	}

	user, err := conn.findUser("id = ?", refreshToken.UserID)
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
func (conn *Connection) IsDuplicatedEmail(instanceID string, email, aud string) (bool, error) {
	_, err := conn.FindUserByEmailAndAudience(instanceID, email, aud)
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
	if result := tx.Save(user); result.Error != nil {
		tx.Rollback()
		return errors.Wrap(result.Error, "Error updating user record")
	}
	return nil
}

// GetInstance finds an instance by ID
func (conn *Connection) GetInstance(instanceID string) (*models.Instance, error) {
	instance := models.Instance{}
	if rsp := conn.db.Where("id = ?", instanceID).First(&instance); rsp.Error != nil {
		if rsp.RecordNotFound() {
			return nil, models.InstanceNotFoundError{}
		}
		return nil, errors.Wrap(rsp.Error, "error finding instance")
	}
	return &instance, nil
}

func (conn *Connection) GetInstanceByUUID(uuid string) (*models.Instance, error) {
	instance := models.Instance{}
	if rsp := conn.db.Where("uuid = ?", uuid).First(&instance); rsp.Error != nil {
		if rsp.RecordNotFound() {
			return nil, models.InstanceNotFoundError{}
		}
		return nil, errors.Wrap(rsp.Error, "error finding instance")
	}
	return &instance, nil
}

func (conn *Connection) CreateInstance(instance *models.Instance) error {
	if result := conn.db.Create(instance); result.Error != nil {
		return errors.Wrap(result.Error, "Error creating instance")
	}
	return nil
}

func (conn *Connection) UpdateInstance(instance *models.Instance) error {
	if result := conn.db.Save(instance); result.Error != nil {
		return errors.Wrap(result.Error, "Error updating instance record")
	}
	return nil
}

func (conn *Connection) DeleteInstance(instance *models.Instance) error {
	tx := conn.db.Begin()

	delModels := map[string]interface{}{
		"user":          models.User{},
		"refresh token": models.RefreshToken{},
	}

	for name, dm := range delModels {
		if result := tx.Delete(dm, "instance_id = ?", instance.ID); result.Error != nil {
			tx.Rollback()
			return errors.Wrap(result.Error, fmt.Sprintf("Error deleting %s records", name))
		}
	}

	if result := tx.Delete(instance); result.Error != nil {
		tx.Rollback()
		return errors.Wrap(result.Error, "Error deleting instance record")
	}

	return tx.Commit().Error
}

// Dial will connect to that storage engine
func Dial(config *conf.GlobalConfiguration) (*Connection, error) {
	if config.DB.Driver == "" && config.DB.URL != "" {
		u, err := url.Parse(config.DB.URL)
		if err != nil {
			return nil, errors.Wrap(err, "parsing db connection url")
		}
		config.DB.Driver = u.Scheme
	}

	if config.DB.Dialect == "" {
		config.DB.Dialect = config.DB.Driver
	}
	db, err := gorm.Open(config.DB.Dialect, config.DB.Driver, config.DB.URL)
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}

	if err := db.DB().Ping(); err != nil {
		return nil, errors.Wrap(err, "checking database connection")
	}

	db.SetLogger(logger{logrus.WithField("db-connection", config.DB.Driver)})

	if logrus.StandardLogger().Level == logrus.DebugLevel {
		db.LogMode(true)
	}

	conn := &Connection{
		db: db,
	}

	return conn, nil
}

func createRefreshToken(tx *gorm.DB, user *models.User) (*models.RefreshToken, error) {
	token := &models.RefreshToken{
		InstanceID: user.InstanceID,
		User:       *user,
		UserID:     user.ID,
		Token:      crypto.SecureToken(),
	}

	if err := tx.Create(token).Error; err != nil {
		return nil, errors.Wrap(err, "error creating refresh token")
	}

	return token, nil
}
