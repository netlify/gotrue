package mongo

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/models"
	"github.com/pkg/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/txn"
)

type logger struct {
	entry *logrus.Entry
}

func (l logger) Output(depth int, s string) error {
	l.entry.Print(s)
	return nil
}

// Connection represents a MongoDB connection.
type Connection struct {
	session *mgo.Session
	db      *mgo.Database
	config  *conf.Configuration
}

// Close closes the mongo connection.
func (conn *Connection) Close() error {
	conn.session.Close()
	return nil
}

// Automigrate is not necessary for mongo.
func (conn *Connection) Automigrate() error {
	return nil
}

// CreateUser creates a user in mongo.
func (conn *Connection) CreateUser(user *models.User) error {
	c := conn.db.C(user.TableName())
	if err := c.Insert(user); err != nil {
		return errors.Wrap(err, "Error creating user")
	}

	return conn.makeUserAdmin(c, user)
}

// DeleteUser deletes a user from mongo.
func (conn *Connection) DeleteUser(user *models.User) error {
	u, err := conn.FindUserByID(user.ID)
	if err != nil {
		return err
	}

	c := conn.db.C(u.TableName())
	return c.Remove(bson.M{"_id": user.ID})
}

// FindUsersInAudience finds users that belong to the provided audience.
func (conn *Connection) FindUsersInAudience(aud string) ([]*models.User, error) {
	user := &models.User{}
	users := []*models.User{}
	c := conn.db.C(user.TableName())
	err := c.Find(bson.M{"aud": aud}).All(&users)
	return users, err
}

func (conn *Connection) findUser(query bson.M) (*models.User, error) {
	user := &models.User{}
	c := conn.db.C(user.TableName())

	if err := c.Find(query).One(user); err != nil {
		if err == mgo.ErrNotFound {
			return nil, models.UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}
	return user, nil
}

// FindUserByConfirmationToken finds a user with the specified confirmation token.
func (conn *Connection) FindUserByConfirmationToken(token string) (*models.User, error) {
	return conn.findUser(bson.M{"confirmation_token": token})
}

// FindUserByEmailAndAudience finds a user with the specified email and audience.
func (conn *Connection) FindUserByEmailAndAudience(email, aud string) (*models.User, error) {
	return conn.findUser(bson.M{"email": email, "aud": aud})
}

// FindUserByID finds a user with specified ID.
func (conn *Connection) FindUserByID(id string) (*models.User, error) {
	return conn.findUser(bson.M{"_id": id})
}

// FindUserByRecoveryToken finds a user with the specified recovery token.
func (conn *Connection) FindUserByRecoveryToken(token string) (*models.User, error) {
	return conn.findUser(bson.M{"recovery_token": token})
}

// FindUserWithRefreshToken finds a user with the specified refresh token.
func (conn *Connection) FindUserWithRefreshToken(token, aud string) (*models.User, *models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	rc := conn.db.C(refreshToken.TableName())

	if err := rc.Find(bson.M{"token": token}).One(refreshToken); err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil, models.RefreshTokenNotFoundError{}
		}
		return nil, nil, errors.Wrap(err, "error finding refresh token")
	}

	user, err := conn.findUser(bson.M{"_id": refreshToken.UserID, "aud": aud})
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// GrantAuthenticatedUser issues a new refresh token for a user.
func (conn *Connection) GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error) {
	runner := conn.newTxRunner()

	token := &models.RefreshToken{
		User:   *user,
		UserID: user.ID,
		Token:  crypto.SecureToken(),
		BID:    bson.NewObjectId(),
	}

	ops := []txn.Op{{
		C:      user.TableName(),
		Id:     user.ID,
		Update: bson.M{"$set": user},
	}, {
		C:      token.TableName(),
		Id:     token.BID,
		Insert: token,
	}}

	if err := runner.Run(ops, bson.NewObjectId(), nil); err != nil {
		return nil, errors.Wrap(err, "error granting authenticated user")
	}

	return token, nil
}

// GrantRefreshTokenSwap swaps an issued refresh token for a new one.
func (conn *Connection) GrantRefreshTokenSwap(user *models.User, token *models.RefreshToken) (*models.RefreshToken, error) {
	runner := conn.newTxRunner()

	token.Revoked = true

	newToken := &models.RefreshToken{
		User:   *user,
		UserID: user.ID,
		Token:  crypto.SecureToken(),
		BID:    bson.NewObjectId(),
	}

	ops := []txn.Op{{
		C:      token.TableName(),
		Id:     token.BID,
		Update: bson.M{"$set": token},
	}, {
		C:      token.TableName(),
		Id:     newToken.BID,
		Insert: newToken,
	}}

	if err := runner.Run(ops, bson.NewObjectId(), nil); err != nil {
		return nil, errors.Wrap(err, "error granting authenticated user")
	}

	return newToken, nil
}

// IsDuplicatedEmail returns whether an email and audience are already in use.
func (conn *Connection) IsDuplicatedEmail(email, aud string) (bool, error) {
	_, err := conn.findUser(bson.M{"email": email, "aud": aud})
	if err != nil {
		if models.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Logout removes all refresh tokens for a user
func (conn *Connection) Logout(id string) {
	t := &models.RefreshToken{}
	c := conn.db.C(t.TableName())
	c.RemoveAll(bson.M{"user_id": id})
}

func (conn *Connection) makeUserAdmin(c *mgo.Collection, user *models.User) error {
	if conn.config.JWT.AdminGroupDisabled {
		return nil
	}

	// Automatically make first user admin
	v, err := c.Find(bson.M{"_id": bson.M{"$ne": user.ID}}).Count()
	if err != nil {
		return errors.Wrap(err, "Error checking existing user count in makeUserAdmin")
	}

	if v == 0 {
		user.SetRole(conn.config.JWT.AdminGroupName)
		if err := c.Update(bson.M{"_id": user.ID}, bson.M{"$set": user}); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error setting administrative privileges for user %s", user.ID))
		}
	}

	return nil
}

// RevokeToken revokes a refresh token.
func (conn *Connection) RevokeToken(token *models.RefreshToken) error {
	token.Revoked = true

	c := conn.db.C(token.TableName())
	if err := c.Update(bson.M{"_id": token.BID}, bson.M{"$set": token}); err != nil {
		return errors.Wrap(err, "error revoking refresh token")
	}
	return nil
}

// RollbackRefreshTokenSwap rolls back a refresh token swap by revoking the new
// token and un-revoking the old token.
func (conn *Connection) RollbackRefreshTokenSwap(newToken, oldToken *models.RefreshToken) error {
	runner := conn.newTxRunner()

	newToken.Revoked = true
	oldToken.Revoked = false

	ops := []txn.Op{{
		C:      newToken.TableName(),
		Id:     newToken.BID,
		Update: bson.M{"$set": newToken},
	}, {
		C:      oldToken.TableName(),
		Id:     oldToken.BID,
		Update: bson.M{"$set": oldToken},
	}}

	if err := runner.Run(ops, bson.NewObjectId(), nil); err != nil {
		return errors.Wrap(err, "error granting authenticated user")
	}

	return nil
}

// UpdateUser updates a user document.
func (conn *Connection) UpdateUser(user *models.User) error {
	c := conn.db.C(user.TableName())
	if err := c.Update(bson.M{"_id": user.ID}, bson.M{"$set": user}); err != nil {
		return errors.Wrap(err, "Error updating user record")
	}
	return nil
}

func (conn *Connection) newTxRunner() *txn.Runner {
	return txn.NewRunner(conn.db.C("auth_transactions"))
}

// Dial will connect to that storage engine
func Dial(config *conf.Configuration) (*Connection, error) {
	mgo.SetLogger(logger{logrus.WithField("db-connection", "mongo")})

	if config.Logging.IsDebugEnabled() {
		mgo.SetDebug(true)
	}

	session, err := mgo.Dial(config.DB.ConnURL)
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}

	if err := session.Ping(); err != nil {
		return nil, errors.Wrap(err, "checking database connection")
	}

	conn := &Connection{
		session: session,
		// Take the database name from the connection URL
		db:     session.DB(""),
		config: config,
	}

	return conn, nil
}
