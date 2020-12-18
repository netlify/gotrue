package models

import (
	"strings"
	"time"

	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const SystemUserID = "0"

var SystemUserUUID = uuid.Nil

func init() {
	storage.AddMigration(&User{})
}

// User represents a registered user with email/password authentication
type User struct {
	InstanceID uuid.UUID `json:"-" gorm:"index:users_instance_id_idx;index:users_instance_id_email_idx;type:varchar(255) DEFAULT NULL"`
	ID         uuid.UUID `json:"id" gorm:"primaryKey;type:varchar(255) NOT NULL"`

	Aud               string     `json:"aud" gorm:"type:varchar(255) DEFAULT NULL"`
	Role              string     `json:"role" gorm:"type:varchar(255) DEFAULT NULL"`
	Email             string     `json:"email" gorm:"index:users_instance_id_email_idx;type:varchar(255) DEFAULT NULL"`
	EncryptedPassword string     `json:"-" gorm:"type:varchar(255) DEFAULT NULL"`
	ConfirmedAt       *time.Time `json:"confirmed_at,omitempty" gorm:"type:timestamp NULL DEFAULT NULL"`
	InvitedAt         *time.Time `json:"invited_at,omitempty" gorm:"type:timestamp NULL DEFAULT NULL"`

	ConfirmationToken  string     `json:"-" gorm:"type:varchar(255) DEFAULT NULL"`
	ConfirmationSentAt *time.Time `json:"confirmation_sent_at,omitempty" gorm:"type:timestamp NULL DEFAULT NULL"`

	RecoveryToken  string     `json:"-" gorm:"type:varchar(255) DEFAULT NULL"`
	RecoverySentAt *time.Time `json:"recovery_sent_at,omitempty" gorm:"type:timestamp NULL DEFAULT NULL"`

	EmailChangeToken  string     `json:"-" gorm:"type:varchar(255) DEFAULT NULL"`
	EmailChange       string     `json:"new_email,omitempty" gorm:"type:varchar(255) DEFAULT NULL"`
	EmailChangeSentAt *time.Time `json:"email_change_sent_at,omitempty" gorm:"type:timestamp NULL DEFAULT NULL"`

	LastSignInAt *time.Time `json:"last_sign_in_at,omitempty" gorm:"type:timestamp NULL DEFAULT NULL"`

	AppMetaData  JSONMap `json:"app_metadata" gorm:"column:raw_app_meta_data;type:JSON NULL DEFAULT NULL"`
	UserMetaData JSONMap `json:"user_metadata" gorm:"column:raw_user_meta_data;type:JSON NULL DEFAULT NULL"`

	IsSuperAdmin bool `json:"-" gorm:"type:tinyint(1) DEFAULT NULL"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp NULL DEFAULT NULL"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamp NULL DEFAULT NULL"`
}

// NewUser initializes a new user from an email, password and user data.
func NewUser(instanceID uuid.UUID, email, password, aud string, userData map[string]interface{}) (*User, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, errors.Wrap(err, "Error generating unique id")
	}
	pw, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &User{
		InstanceID:        instanceID,
		ID:                id,
		Aud:               aud,
		Email:             email,
		UserMetaData:      userData,
		EncryptedPassword: pw,
	}
	return user, nil
}

func NewSystemUser(instanceID uuid.UUID, aud string) *User {
	return &User{
		InstanceID:   instanceID,
		ID:           SystemUserUUID,
		Aud:          aud,
		IsSuperAdmin: true,
	}
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	return u.BeforeUpdate(tx)
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if u.ID == SystemUserUUID {
		return errors.New("Cannot persist system user")
	}

	return nil
}

func (u *User) BeforeSave(tx *gorm.DB) error {
	if u.ID == SystemUserUUID {
		return errors.New("Cannot persist system user")
	}

	if u.ConfirmedAt != nil && u.ConfirmedAt.IsZero() {
		u.ConfirmedAt = nil
	}
	if u.InvitedAt != nil && u.InvitedAt.IsZero() {
		u.InvitedAt = nil
	}
	if u.ConfirmationSentAt != nil && u.ConfirmationSentAt.IsZero() {
		u.ConfirmationSentAt = nil
	}
	if u.RecoverySentAt != nil && u.RecoverySentAt.IsZero() {
		u.RecoverySentAt = nil
	}
	if u.EmailChangeSentAt != nil && u.EmailChangeSentAt.IsZero() {
		u.EmailChangeSentAt = nil
	}
	if u.LastSignInAt != nil && u.LastSignInAt.IsZero() {
		u.LastSignInAt = nil
	}
	return nil
}

// IsConfirmed checks if a user has already being
// registered and confirmed.
func (u *User) IsConfirmed() bool {
	return u.ConfirmedAt != nil
}

// SetRole sets the users Role to roleName
func (u *User) SetRole(tx *storage.Connection, roleName string) error {
	u.Role = strings.TrimSpace(roleName)
	return tx.Model(&u).Select("role").Updates(u).Error
}

// HasRole returns true when the users role is set to roleName
func (u *User) HasRole(roleName string) bool {
	return u.Role == roleName
}

// UpdateUserMetaData sets all user data from a map of updates,
// ensuring that it doesn't override attributes that are not
// in the provided map.
func (u *User) UpdateUserMetaData(tx *storage.Connection, updates map[string]interface{}) error {
	if u.UserMetaData == nil {
		u.UserMetaData = updates
	} else if updates != nil {
		for key, value := range updates {
			if value != nil {
				u.UserMetaData[key] = value
			} else {
				delete(u.UserMetaData, key)
			}
		}
	}
	return tx.Model(&u).Select("raw_user_meta_data").Updates(u).Error
}

// UpdateAppMetaData updates all app data from a map of updates
func (u *User) UpdateAppMetaData(tx *storage.Connection, updates map[string]interface{}) error {
	if u.AppMetaData == nil {
		u.AppMetaData = updates
	} else if updates != nil {
		for key, value := range updates {
			if value != nil {
				u.AppMetaData[key] = value
			} else {
				delete(u.AppMetaData, key)
			}
		}
	}
	return tx.Model(&u).Select("raw_app_meta_data").Updates(u).Error
}

func (u *User) SetEmail(tx *storage.Connection, email string) error {
	u.Email = email
	return tx.Model(&u).Select("email").Updates(u).Error
}

// hashPassword generates a hashed password from a plaintext string
func hashPassword(password string) (string, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

func (u *User) UpdatePassword(tx *storage.Connection, password string) error {
	pw, err := hashPassword(password)
	if err != nil {
		return err
	}
	u.EncryptedPassword = pw
	return tx.Model(&u).Select("encrypted_password").Updates(u).Error
}

// Authenticate a user from a password
func (u *User) Authenticate(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password))
	return err == nil
}

// Confirm resets the confimation token and the confirm timestamp
func (u *User) Confirm(tx *storage.Connection) error {
	u.ConfirmationToken = ""
	now := time.Now()
	u.ConfirmedAt = &now
	return tx.Model(&u).Select("confirmation_token", "confirmed_at").Updates(u).Error
}

// ConfirmEmailChange confirm the change of email for a user
func (u *User) ConfirmEmailChange(tx *storage.Connection) error {
	u.Email = u.EmailChange
	u.EmailChange = ""
	u.EmailChangeToken = ""
	return tx.Model(&u).Select("email", "email_change", "email_change_token").Updates(u).Error
}

// Recover resets the recovery token
func (u *User) Recover(tx *storage.Connection) error {
	u.RecoveryToken = ""
	return tx.Model(&u).Select("recovery_token").Updates(u).Error
}

// CountOtherUsers counts how many other users exist besides the one provided
func CountOtherUsers(tx *storage.Connection, instanceID, id uuid.UUID) (int, error) {
	var userCount int64
	err := tx.Model(&User{}).Where("instance_id = ? and id != ?", instanceID, id).Count(&userCount).Error
	return int(userCount), errors.Wrap(err, "error finding registered users")
}

func findUser(tx *storage.Connection, query string, args ...interface{}) (*User, error) {
	obj := &User{}
	if err := tx.Where(query, args...).First(obj).Error; err != nil {
		if errors.Cause(err) == gorm.ErrRecordNotFound {
			return nil, UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}

	return obj, nil
}

// FindUserByConfirmationToken finds users with the matching confirmation token.
func FindUserByConfirmationToken(tx *storage.Connection, token string) (*User, error) {
	return findUser(tx, "confirmation_token = ?", token)
}

// FindUserByEmailAndAudience finds a user with the matching email and audience.
func FindUserByEmailAndAudience(tx *storage.Connection, instanceID uuid.UUID, email, aud string) (*User, error) {
	return findUser(tx, "instance_id = ? and email = ? and aud = ?", instanceID, email, aud)
}

// FindUserByID finds a user matching the provided ID.
func FindUserByID(tx *storage.Connection, id uuid.UUID) (*User, error) {
	return findUser(tx, "id = ?", id)
}

// FindUserByInstanceIDAndID finds a user matching the provided ID.
func FindUserByInstanceIDAndID(tx *storage.Connection, instanceID, id uuid.UUID) (*User, error) {
	return findUser(tx, "instance_id = ? and id = ?", instanceID, id)
}

// FindUserByRecoveryToken finds a user with the matching recovery token.
func FindUserByRecoveryToken(tx *storage.Connection, token string) (*User, error) {
	return findUser(tx, "recovery_token = ?", token)
}

// FindUserWithRefreshToken finds a user from the provided refresh token.
func FindUserWithRefreshToken(tx *storage.Connection, token string) (*User, *RefreshToken, error) {
	refreshToken := &RefreshToken{}
	if err := tx.First(refreshToken, "token = ?", token).Error; err != nil {
		if errors.Cause(err) == gorm.ErrRecordNotFound {
			return nil, nil, RefreshTokenNotFoundError{}
		}
		return nil, nil, errors.Wrap(err, "error finding refresh token")
	}

	user, err := findUser(tx, "id = ?", refreshToken.UserID)
	if err != nil {
		return nil, nil, err
	}

	return user, refreshToken, nil
}

// FindUsersInAudience finds users with the matching audience.
func FindUsersInAudience(tx *storage.Connection, instanceID uuid.UUID, aud string, pageParams *Pagination, sortParams *SortParams, filter string) ([]*User, error) {
	users := []*User{}
	q := tx.Model(users).Where("instance_id = ? and aud = ?", instanceID, aud)

	if filter != "" {
		lf := "%" + filter + "%"
		// we must specify the collation in order to get case insensitive search for the JSON column
		q = q.Where("(email LIKE ? OR raw_user_meta_data->>'$.full_name' COLLATE utf8mb4_unicode_ci LIKE ?)", lf, lf)
	}

	if sortParams != nil && len(sortParams.Fields) > 0 {
		for _, field := range sortParams.Fields {
			q = q.Order(field.Name + " " + string(field.Dir))
		}
	}

	var err error
	if pageParams != nil {
		p := paginator.New(adapter.NewGORMAdapter(q), int(pageParams.PerPage))
		p.SetPage(int(pageParams.Page))
		if err = p.Results(&users); err != nil {
			return nil, err
		}
		var cnt int
		if cnt, err = p.PageNums(); err != nil {
			return nil, err
		}
		pageParams.Count = uint64(cnt)
	} else {
		err = q.Find(&users).Error
	}

	return users, err
}

// IsDuplicatedEmail returns whether a user exists with a matching email and audience.
func IsDuplicatedEmail(tx *storage.Connection, instanceID uuid.UUID, email, aud string) (bool, error) {
	_, err := FindUserByEmailAndAudience(tx, instanceID, email, aud)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
