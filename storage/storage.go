package storage

import (
	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/models"
	"github.com/netlify/netlify-auth/storage/mongo"
	"github.com/netlify/netlify-auth/storage/sql"
)

type Connection interface {
	Close() error
	Automigrate(models ...interface{}) error
	CreateUser(user *models.User) error
	FindUserByEmail(email string) (*models.User, error)
	FindUserByID(id string) (*models.User, error)
	FindUserByVerificationToken(verificationType models.VerifyType, token string) (*models.User, error)
	FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error)
	GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error)
	GrantRefreshTokenSwap(user *models.User, token *models.RefreshToken) (*models.RefreshToken, error)
	IsDuplicatedEmail(email, id string) (bool, error)
	Logout(id interface{})
	RevokeToken(token *models.RefreshToken) error
	RollbackRefreshTokenSwap(newToken, oldToken *models.RefreshToken) error
	UpdateUser(user *models.User) error
}

// Connect will connect to that storage engine
func Connect(config *conf.Configuration) (Connection, error) {
	if config.DB.Namespace != "" {
		models.Namespace = config.DB.Namespace
	}

	if config.DB.Driver == "mongo" {
		return mongo.Connect(config)
	}

	return sql.Connect(config)
}
