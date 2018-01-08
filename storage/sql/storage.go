package sql

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"

	// import drivers we might need
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	_ "github.com/go-sql-driver/mysql"
	uuid "github.com/satori/go.uuid"

	"github.com/markbates/pop"
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
	db *pop.Connection
}

// Close closes the database connection.
func (conn *Connection) Close() error {
	return conn.db.Close()
}

// CreateUser creates a user.
func (conn *Connection) CreateUser(user *models.User) error {
	return conn.db.Transaction(func(tx *pop.Connection) error {
		return tx.Create(user)
	})
}

// CountOtherUsers counts how many other users exist besides the one provided
func (conn *Connection) CountOtherUsers(instanceID, id uuid.UUID) (int, error) {
	userCount, err := conn.db.Q().Where("instance_id = ? and id != ?", instanceID, id).Count(&models.User{})
	return userCount, errors.Wrap(err, "error finding registered users")
}

func (conn *Connection) findUser(query string, args ...interface{}) (*models.User, error) {
	obj := &models.User{}
	if err := conn.db.Q().Where(query, args...).First(obj); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, models.UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}

	return obj, nil
}

// DeleteUser deletes a user.
func (conn *Connection) DeleteUser(u *models.User) error {
	return conn.db.Destroy(u)
}

// FindUsersInAudience finds users with the matching audience.
func (conn *Connection) FindUsersInAudience(instanceID uuid.UUID, aud string, pageParams *models.Pagination, sortParams *models.SortParams, filter string) ([]*models.User, error) {
	users := []*models.User{}
	q := conn.db.Q().Where("instance_id = ? and aud = ?", instanceID, aud)

	if filter != "" {
		lf := "%" + filter + "%"
		// we must specify the collation in order to get case insensitive search for the JSON column
		q = q.Where("email LIKE ? OR raw_user_meta_data->>'$.full_name' COLLATE utf8mb4_unicode_ci LIKE ?", lf, lf)
	}

	if sortParams != nil && len(sortParams.Fields) > 0 {
		for _, field := range sortParams.Fields {
			q = q.Order(field.Name + " " + string(field.Dir))
		}
	}

	var err error
	if pageParams != nil {
		err = q.Paginate(int(pageParams.Page), int(pageParams.PerPage)).All(&users)
		pageParams.Count = uint64(q.Paginator.TotalEntriesSize)
	} else {
		err = q.All(&users)
	}

	return users, err
}

// FindUserByConfirmationToken finds users with the matching confirmation token.
func (conn *Connection) FindUserByConfirmationToken(token string) (*models.User, error) {
	return conn.findUser("confirmation_token = ?", token)
}

// FindUserByEmailAndAudience finds a user with the matching email and audience.
func (conn *Connection) FindUserByEmailAndAudience(instanceID uuid.UUID, email, aud string) (*models.User, error) {
	return conn.findUser("instance_id = ? and email = ? and aud = ?", instanceID, email, aud)
}

// FindUserByID finds a user matching the provided ID.
func (conn *Connection) FindUserByID(id uuid.UUID) (*models.User, error) {
	return conn.findUser("id = ?", id)
}

// FindUserByInstanceIDAndID finds a user matching the provided ID.
func (conn *Connection) FindUserByInstanceIDAndID(instanceID, id uuid.UUID) (*models.User, error) {
	return conn.findUser("instance_id = ? and id = ?", instanceID, id)
}

// FindUserByRecoveryToken finds a user with the matching recovery token.
func (conn *Connection) FindUserByRecoveryToken(token string) (*models.User, error) {
	return conn.findUser("recovery_token = ?", token)
}

// FindUserWithRefreshToken finds a user from the provided refresh token.
func (conn *Connection) FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	if err := conn.db.Where("token = ?", token).First(refreshToken); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil, models.RefreshTokenNotFoundError{}
		}
		return nil, nil, errors.Wrap(err, "error finding refresh token")
	}

	user, err := conn.findUser("id = ?", refreshToken.UserID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// GrantAuthenticatedUser creates a refresh token for the provided user.
func (conn *Connection) GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error) {
	var token *models.RefreshToken
	err := conn.db.Transaction(func(tx *pop.Connection) error {
		terr := tx.Save(user)
		if terr != nil {
			return terr
		}
		token, terr = createRefreshToken(tx, user)
		return terr
	})
	return token, err
}

// GrantRefreshTokenSwap swaps a refresh token for a new one, revoking the provided token.
func (conn *Connection) GrantRefreshTokenSwap(user *models.User, token *models.RefreshToken) (*models.RefreshToken, error) {
	var newToken *models.RefreshToken
	err := conn.db.Transaction(func(tx *pop.Connection) error {
		terr := revokeToken(tx, token, true)
		if terr != nil {
			return terr
		}
		newToken, terr = createRefreshToken(tx, user)
		return terr
	})
	return newToken, err
}

// IsDuplicatedEmail returns whether a user exists with a matching email and audience.
func (conn *Connection) IsDuplicatedEmail(instanceID uuid.UUID, email, aud string) (bool, error) {
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
func (conn *Connection) Logout(id uuid.UUID) {
	conn.db.RawQuery("DELETE FROM "+(&pop.Model{Value: models.RefreshToken{}}).TableName()+" WHERE user_id = ?", id).Exec()
}

func revokeToken(tx *pop.Connection, token *models.RefreshToken, revoked bool) error {
	token.Revoked = revoked
	return tx.Update(token, "instance_id", "token", "user_id")
}

// RevokeToken revokes a refresh token.
func (conn *Connection) RevokeToken(token *models.RefreshToken) error {
	return conn.db.Transaction(func(tx *pop.Connection) error {
		return revokeToken(tx, token, true)
	})
}

// RollbackRefreshTokenSwap rolls back a refresh token swap by revoking the new
// token, and un-revoking the old token.
func (conn *Connection) RollbackRefreshTokenSwap(newToken, oldToken *models.RefreshToken) error {
	return conn.db.Transaction(func(tx *pop.Connection) error {
		if err := revokeToken(tx, newToken, true); err != nil {
			return err
		}
		return revokeToken(tx, oldToken, false)
	})
}

// UpdateUser updates a user.
func (conn *Connection) UpdateUser(user *models.User) error {
	return errors.Wrap(conn.db.Transaction(func(tx *pop.Connection) error {
		return tx.Update(user)
	}), "Error updating user record")
}

// GetInstance finds an instance by ID
func (conn *Connection) GetInstance(instanceID uuid.UUID) (*models.Instance, error) {
	instance := models.Instance{}
	if err := conn.db.Find(&instance, instanceID); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, models.InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return &instance, nil
}

func (conn *Connection) GetInstanceByUUID(uuid uuid.UUID) (*models.Instance, error) {
	instance := models.Instance{}
	if err := conn.db.Where("uuid = ?", uuid).First(&instance); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, models.InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return &instance, nil
}

func (conn *Connection) CreateInstance(instance *models.Instance) error {
	return errors.Wrap(conn.db.Create(instance), "Error creating instance")
}

func (conn *Connection) UpdateInstance(instance *models.Instance) error {
	return errors.Wrap(conn.db.Update(instance), "Error updating instance record")
}

func (conn *Connection) DeleteInstance(instance *models.Instance) error {
	return conn.db.Transaction(func(tx *pop.Connection) error {
		delModels := map[string]*pop.Model{
			"user":          &pop.Model{Value: models.User{}},
			"refresh token": &pop.Model{Value: models.RefreshToken{}},
		}

		for name, dm := range delModels {
			if err := tx.RawQuery("DELETE FROM "+dm.TableName()+" WHERE instance_id = ?", instance.ID).Exec(); err != nil {
				return errors.Wrapf(err, "Error deleting %s records", name)
			}
		}

		return errors.Wrap(tx.Destroy(instance), "Error deleting instance record")
	})
}

func (conn *Connection) TruncateAll() error {
	return conn.db.Transaction(func(tx *pop.Connection) error {
		if err := tx.RawQuery("TRUNCATE " + (&pop.Model{Value: models.User{}}).TableName()).Exec(); err != nil {
			return err
		}
		if err := tx.RawQuery("TRUNCATE " + (&pop.Model{Value: models.RefreshToken{}}).TableName()).Exec(); err != nil {
			return err
		}
		return tx.RawQuery("TRUNCATE " + (&pop.Model{Value: models.Instance{}}).TableName()).Exec()
	})
}

func (conn *Connection) MigrateUp() error {
	p, err := filepath.Abs("../migrations")
	if err != nil {
		return err
	}
	fmt.Println(p)
	fm, err := pop.NewFileMigrator(p, conn.db)
	if err != nil {
		return err
	}
	fm.SchemaPath = ""
	return fm.Up()
}

func (conn *Connection) DropDB() error {
	return pop.DropDB(conn.db)
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

	db, err := pop.NewConnection(&pop.ConnectionDetails{
		Dialect: config.DB.Driver,
		URL:     config.DB.URL,
	})
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}
	if err := db.Open(); err != nil {
		return nil, errors.Wrap(err, "checking database connection")
	}

	if config.DB.Namespace != "" {
		pop.MapTableName("User", config.DB.Namespace+"_users")
		pop.MapTableName("RefreshToken", config.DB.Namespace+"_refresh_tokens")
		pop.MapTableName("Instance", config.DB.Namespace+"_instances")
	}

	if logrus.StandardLogger().Level == logrus.DebugLevel {
		pop.Debug = true
	}

	conn := &Connection{
		db: db,
	}

	return conn, nil
}

func createRefreshToken(tx *pop.Connection, user *models.User) (*models.RefreshToken, error) {
	token := &models.RefreshToken{
		InstanceID: user.InstanceID,
		UserID:     user.ID,
		Token:      crypto.SecureToken(),
	}

	if err := tx.Create(token); err != nil {
		return nil, errors.Wrap(err, "error creating refresh token")
	}

	return token, nil
}
