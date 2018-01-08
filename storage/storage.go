package storage

import (
	"github.com/netlify/gotrue/models"
	uuid "github.com/satori/go.uuid"
)

// Connection is the interface a storage provider must implement.
type Connection interface {
	Close() error
	TruncateAll() error
	DropDB() error
	MigrateUp() error
	CountOtherUsers(instanceID uuid.UUID, id uuid.UUID) (int, error)
	CreateUser(user *models.User) error
	DeleteUser(user *models.User) error
	UpdateUser(user *models.User) error
	FindUserByConfirmationToken(token string) (*models.User, error)
	FindUserByEmailAndAudience(instanceID uuid.UUID, email, aud string) (*models.User, error)
	FindUserByID(id uuid.UUID) (*models.User, error)
	FindUserByInstanceIDAndID(instanceID, id uuid.UUID) (*models.User, error)
	FindUserByRecoveryToken(token string) (*models.User, error)
	FindUserWithRefreshToken(token string) (*models.User, *models.RefreshToken, error)
	FindUsersInAudience(instanceID uuid.UUID, aud string, pageParams *models.Pagination, sortParams *models.SortParams, filter string) ([]*models.User, error)
	GrantAuthenticatedUser(user *models.User) (*models.RefreshToken, error)
	GrantRefreshTokenSwap(user *models.User, token *models.RefreshToken) (*models.RefreshToken, error)
	IsDuplicatedEmail(instanceID uuid.UUID, email, aud string) (bool, error)
	Logout(id uuid.UUID)
	RevokeToken(token *models.RefreshToken) error
	RollbackRefreshTokenSwap(newToken, oldToken *models.RefreshToken) error

	GetInstanceByUUID(uuid uuid.UUID) (*models.Instance, error)
	GetInstance(instanceID uuid.UUID) (*models.Instance, error)
	CreateInstance(instance *models.Instance) error
	DeleteInstance(instance *models.Instance) error
	UpdateInstance(instance *models.Instance) error
}
