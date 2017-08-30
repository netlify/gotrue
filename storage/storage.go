package storage

import "github.com/netlify/gotrue/models"

// Connection is the interface a storage provider must implement.
type Connection interface {
	Close() error
	Automigrate() error
	CountOtherUsers(instanceID string, id string) (int, error)
	CreateUser(user *models.User) error
	DeleteUser(user *models.User) error
	UpdateUser(user *models.User) error
	FindUserByConfirmationToken(token string) (*models.User, error)
	FindUserByEmailAndAudience(instanceID string, email, aud string) (*models.User, error)
	FindUserByID(id string) (*models.User, error)
	FindUserByInstanceIDAndID(instanceID, id string) (*models.User, error)
	FindUserByRecoveryToken(token string) (*models.User, error)
	FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error)
	FindUsersInAudience(instanceID string, aud string, pageParams *models.Pagination, sortParams *models.SortParams) ([]*models.User, error)
	GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error)
	GrantRefreshTokenSwap(user *models.User, token *models.RefreshToken) (*models.RefreshToken, error)
	IsDuplicatedEmail(instanceID string, email, aud string) (bool, error)
	Logout(id string)
	RevokeToken(token *models.RefreshToken) error
	RollbackRefreshTokenSwap(newToken, oldToken *models.RefreshToken) error

	GetInstanceByUUID(uuid string) (*models.Instance, error)
	GetInstance(instanceID string) (*models.Instance, error)
	CreateInstance(instance *models.Instance) error
	DeleteInstance(instance *models.Instance) error
	UpdateInstance(instance *models.Instance) error
}
