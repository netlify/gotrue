package mongo

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	config  *conf.GlobalConfiguration
}

// Close closes the mongo connection.
func (conn *Connection) Close() error {
	conn.session.Close()
	return nil
}

// Automigrate creates all necessary indexes for mongo.
func (conn *Connection) Automigrate() error {
	collections := []string{
		(&models.RefreshToken{}).TableName(),
		(&models.User{}).TableName(),
	}
	for _, name := range collections {
		c := conn.db.C(name)
		if err := c.EnsureIndex(mgo.Index{
			Key:        []string{"instance_id"},
			Background: true,
		}); err != nil {
			return err
		}
	}

	return nil
}

// CreateUser creates a user in mongo.
func (conn *Connection) CreateUser(user *models.User) error {
	c := conn.db.C(user.TableName())
	if err := c.Insert(user); err != nil {
		return errors.Wrap(err, "Error creating user")
	}
	return nil
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
func (conn *Connection) FindUsersInAudience(instanceID string, aud string, pageParams *models.Pagination) ([]*models.User, error) {
	user := &models.User{}
	users := []*models.User{}
	c := conn.db.C(user.TableName())
	q := c.Find(bson.M{"instance_id": instanceID, "aud": aud})

	var err error
	if pageParams != nil {
		count, err := q.Count()
		if err != nil {
			return nil, err
		}
		pageParams.Count = uint64(count)

		err = q.Skip(int(pageParams.Offset())).Limit(int(pageParams.PerPage)).All(&users)
	} else {
		err = q.All(&users)
	}
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
func (conn *Connection) FindUserByEmailAndAudience(instanceID string, email, aud string) (*models.User, error) {
	return conn.findUser(bson.M{"instance_id": instanceID, "email": email, "aud": aud})
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
func (conn *Connection) FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error) {
	refreshToken := &models.RefreshToken{}
	rc := conn.db.C(refreshToken.TableName())

	if err := rc.Find(bson.M{"token": token}).One(refreshToken); err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil, models.RefreshTokenNotFoundError{}
		}
		return nil, nil, errors.Wrap(err, "error finding refresh token")
	}

	user, err := conn.findUser(bson.M{"_id": refreshToken.UserID})
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// GrantAuthenticatedUser issues a new refresh token for a user.
func (conn *Connection) GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error) {
	runner := conn.newTxRunner()

	token := &models.RefreshToken{
		InstanceID: user.InstanceID,
		User:       *user,
		UserID:     user.ID,
		Token:      crypto.SecureToken(),
		BID:        bson.NewObjectId(),
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
		InstanceID: user.InstanceID,
		User:       *user,
		UserID:     user.ID,
		Token:      crypto.SecureToken(),
		BID:        bson.NewObjectId(),
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

// Logout removes all refresh tokens for a user
func (conn *Connection) Logout(id string) {
	t := &models.RefreshToken{}
	c := conn.db.C(t.TableName())
	c.RemoveAll(bson.M{"user_id": id})
}

// CountOtherUsers counts number of users not matching the id
func (conn *Connection) CountOtherUsers(instanceID string, id string) (int, error) {
	user := models.User{}
	c := conn.db.C(user.TableName())
	return c.Find(bson.M{"instance_id": instanceID, "_id": bson.M{"$ne": id}}).Count()
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

// GetInstance finds an instance by ID
func (conn *Connection) GetInstance(instanceID string) (*models.Instance, error) {
	instance := &models.Instance{}
	c := conn.db.C(instance.TableName())

	if err := c.Find(bson.M{"_id": instanceID}).One(instance); err != nil {
		if err == mgo.ErrNotFound {
			return nil, models.InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return instance, nil
}

func (conn *Connection) GetInstanceByUUID(uuid string) (*models.Instance, error) {
	instance := &models.Instance{}
	c := conn.db.C(instance.TableName())

	if err := c.Find(bson.M{"uuid": uuid}).One(instance); err != nil {
		if err == mgo.ErrNotFound {
			return nil, models.InstanceNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding instance")
	}
	return instance, nil
}

func (conn *Connection) CreateInstance(instance *models.Instance) error {
	c := conn.db.C(instance.TableName())
	if err := c.Insert(instance); err != nil {
		return errors.Wrap(err, "Error creating instance")
	}
	return nil
}

func (conn *Connection) UpdateInstance(instance *models.Instance) error {
	c := conn.db.C(instance.TableName())
	if err := c.Update(bson.M{"_id": instance.ID}, bson.M{"$set": instance}); err != nil {
		return errors.Wrap(err, "Error updating instance record")
	}
	return nil
}

func (conn *Connection) DeleteInstance(instance *models.Instance) error {
	c := conn.db.C(instance.TableName())
	return c.Remove(bson.M{"_id": instance.ID})
}

func (conn *Connection) newTxRunner() *txn.Runner {
	return txn.NewRunner(conn.db.C("auth_transactions"))
}

// Dial will connect to that storage engine
func Dial(config *conf.GlobalConfiguration) (*Connection, error) {
	mgo.SetLogger(logger{logrus.WithField("db-connection", "mongo")})

	if logrus.StandardLogger().Level == logrus.DebugLevel {
		mgo.SetDebug(true)
	}

	session, err := mgo.Dial(config.DB.URL)
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
