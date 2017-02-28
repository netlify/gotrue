package storage

import "github.com/netlify/netlify-auth/models"

type Connection interface {
	Close() error
	Automigrate() error
	CreateUser(user *models.User) error
	FindUserByConfirmationToken(token string) (*models.User, error)
	FindUserByEmailAndAudience(email, aud string) (*models.User, error)
	FindUserByID(id string) (*models.User, error)
	FindUserByRecoveryToken(token string) (*models.User, error)
	FindUserWithRefreshToken(token, aud string) (*models.User, *models.RefreshToken, error)
	GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error)
	GrantRefreshTokenSwap(user *models.User, token *models.RefreshToken) (*models.RefreshToken, error)
	IsDuplicatedEmail(email, aud, id string) (bool, error)
	Logout(id string)
	RevokeToken(token *models.RefreshToken) error
	RollbackRefreshTokenSwap(newToken, oldToken *models.RefreshToken) error
	UpdateUser(user *models.User) error
}
