package models

import (
	"database/sql"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/namespace"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const SystemUserID = "0"

var SystemUserUUID = uuid.Nil

// User respresents a registered user with email/password authentication
type User struct {
	InstanceID uuid.UUID `json:"-" db:"instance_id"`
	ID         uuid.UUID `json:"id" db:"id"`

	Aud               string     `json:"aud" db:"aud"`
	Role              string     `json:"role" db:"role"`
	Email             string     `json:"email" db:"email"`
	EncryptedPassword string     `json:"-" db:"encrypted_password"`
	ConfirmedAt       *time.Time `json:"confirmed_at,omitempty" db:"confirmed_at"`
	InvitedAt         *time.Time `json:"invited_at,omitempty" db:"invited_at"`

	ConfirmationToken  string     `json:"-" db:"confirmation_token"`
	ConfirmationSentAt *time.Time `json:"confirmation_sent_at,omitempty" db:"confirmation_sent_at"`

	RecoveryToken  string     `json:"-" db:"recovery_token"`
	RecoverySentAt *time.Time `json:"recovery_sent_at,omitempty" db:"recovery_sent_at"`

	EmailChangeToken  string     `json:"-" db:"email_change_token"`
	EmailChange       string     `json:"new_email,omitempty" db:"email_change"`
	EmailChangeSentAt *time.Time `json:"email_change_sent_at,omitempty" db:"email_change_sent_at"`

	LastSignInAt *time.Time `json:"last_sign_in_at,omitempty" db:"last_sign_in_at"`

	AppMetaData  JSONMap `json:"app_metadata" db:"raw_app_meta_data"`
	UserMetaData JSONMap `json:"user_metadata" db:"raw_user_meta_data"`

	IsSuperAdmin bool `json:"-" db:"is_super_admin"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
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

func (User) TableName() string {
	tableName := "users"

	if namespace.GetNamespace() != "" {
		return namespace.GetNamespace() + "_" + tableName
	}

	return tableName
}

func (u *User) BeforeCreate(tx *pop.Connection) error {
	return u.BeforeUpdate(tx)
}

func (u *User) BeforeUpdate(tx *pop.Connection) error {
	if u.ID == SystemUserUUID {
		return errors.New("Cannot persist system user")
	}

	return nil
}

func (u *User) BeforeSave(tx *pop.Connection) error {
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
	return tx.UpdateOnly(u, "role")
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
	return tx.UpdateOnly(u, "raw_user_meta_data")
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
	return tx.UpdateOnly(u, "raw_app_meta_data")
}

func (u *User) SetEmail(tx *storage.Connection, email string) error {
	u.Email = email
	return tx.UpdateOnly(u, "email")
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
	return tx.UpdateOnly(u, "encrypted_password")
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
	return tx.UpdateOnly(u, "confirmation_token", "confirmed_at")
}

// ConfirmEmailChange confirm the change of email for a user
func (u *User) ConfirmEmailChange(tx *storage.Connection) error {
	u.Email = u.EmailChange
	u.EmailChange = ""
	u.EmailChangeToken = ""
	return tx.UpdateOnly(u, "email", "email_change", "email_change_token")
}

// Recover resets the recovery token
func (u *User) Recover(tx *storage.Connection) error {
	u.RecoveryToken = ""
	return tx.UpdateOnly(u, "recovery_token")
}

// CountOtherUsers counts how many other users exist besides the one provided
func CountOtherUsers(tx *storage.Connection, instanceID, id uuid.UUID) (int, error) {
	userCount, err := tx.Q().Where("instance_id = ? and id != ?", instanceID, id).Count(&User{})
	return userCount, errors.Wrap(err, "error finding registered users")
}

func findUser(tx *storage.Connection, query string, args ...interface{}) (*User, error) {
	obj := &User{}
	if err := tx.Q().Where(query, args...).First(obj); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
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
	if err := tx.Where("token = ?", token).First(refreshToken); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
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
	q := tx.Q().Where("instance_id = ? and aud = ?", instanceID, aud)

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
		err = q.Paginate(int(pageParams.Page), int(pageParams.PerPage)).All(&users)
		pageParams.Count = uint64(q.Paginator.TotalEntriesSize)
	} else {
		err = q.All(&users)
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
