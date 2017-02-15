package mongo

import (
	"github.com/Sirupsen/logrus"
	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/crypto"
	"github.com/netlify/netlify-auth/models"
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

type Connection struct {
	session *mgo.Session
	db      *mgo.Database
	config  *conf.Configuration
}

func (conn *Connection) Close() error {
	conn.session.Close()
	return nil
}

func (conn *Connection) Automigrate(models ...interface{}) error {
	return nil
}

func (conn *Connection) CreateUser(user *models.User) error {
	c := conn.db.C(user.TableName())
	if err := c.Insert(user); err != nil {
		return errors.Wrap(err, "Error creating user")
	}

	if !conn.config.JWT.AdminGroupDisabled {
		v, err := c.Find(bson.M{"_id": bson.M{"$ne": user.ID}}).Count()
		if err != nil {
			return errors.Wrap(err, "Error making user an admin")
		}

		if v == 0 {
			user.SetRole(conn.config.JWT.AdminGroupName)
			if err := c.Update(bson.M{"_id": user.ID}, bson.M{"$set": user}); err != nil {
				return errors.Wrap(err, "Error making user an admin")
			}
		}
	}

	return nil
}

func (conn *Connection) findUser(query bson.M) (*models.User, error) {
	user := &models.User{}
	c := conn.db.C(user.TableName())

	if err := c.Find(query).One(user); err != nil {
		if err == mgo.ErrNotFound {
			return nil, models.UserNotFoundError{}
		} else {
			return nil, errors.Wrap(err, "error finding user")
		}
	}
	return user, nil
}

func (conn *Connection) FindUserByConfirmationToken(token string) (*models.User, error) {
	return conn.findUser(bson.M{"confirmation_token": token})
}

func (conn *Connection) FindUserByEmail(email string) (*models.User, error) {
	return conn.findUser(bson.M{"email": email})
}

func (conn *Connection) FindUserByID(id string) (*models.User, error) {
	return conn.findUser(bson.M{"_id": id})
}

func (conn *Connection) FindUserByRecoveryToken(token string) (*models.User, error) {
	return conn.findUser(bson.M{"recovery_token": token})
}

func (conn *Connection) FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	rc := conn.db.C(refreshToken.TableName())

	if err := rc.Find(bson.M{"token": token}).One(refreshToken); err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil, models.RefreshTokenNotFoundError{}
		} else {
			return nil, nil, errors.Wrap(err, "error finding refresh token")
		}
	}

	user, err := conn.findUser(bson.M{"_id": refreshToken.UserID})
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

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

	return token, nil
}

func (conn *Connection) IsDuplicatedEmail(email, id string) (bool, error) {
	_, err := conn.findUser(bson.M{"email": email, "_id": bson.M{"$ne": id}})
	if err != nil {
		if err == mgo.ErrNotFound {
			return false, nil
		} else {
			return false, errors.Wrap(err, "error checking duplicated email")
		}
	}

	return true, nil
}

func (conn *Connection) Logout(id interface{}) {
	t := &models.RefreshToken{}
	c := conn.db.C(t.TableName())
	c.RemoveAll(bson.M{"user_id": id})
}

func (conn *Connection) RevokeToken(token *models.RefreshToken) error {
	token.Revoked = true

	c := conn.db.C(token.TableName())
	if err := c.Update(bson.M{"_id": token.BID}, bson.M{"$set": token}); err != nil {
		return errors.Wrap(err, "error revoking refresh token")
	}
	return nil
}

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

// Connect will connect to that storage engine
func Connect(config *conf.Configuration) (*Connection, error) {
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
