package models

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/netlify/gotrue/crypto"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/bcrypt"
)

const SystemUserID = "0"

// User respresents a registered user with email/password authentication
type User struct {
	InstanceID string `json:"-"`
	ID         string `json:"id"`

	Aud               string     `json:"aud"`
	Role              string     `json:"role"`
	Email             string     `json:"email"`
	EncryptedPassword string     `json:"-"`
	ConfirmedAt       *time.Time `json:"confirmed_at,omitempty"`
	InvitedAt         *time.Time `json:"invited_at,omitempty"`

	ConfirmationToken  string     `json:"-"`
	ConfirmationSentAt *time.Time `json:"confirmation_sent_at,omitempty"`

	RecoveryToken  string     `json:"-"`
	RecoverySentAt *time.Time `json:"recovery_sent_at,omitempty"`

	EmailChangeToken  string     `json:"-"`
	EmailChange       string     `json:"new_email,omitempty"`
	EmailChangeSentAt *time.Time `json:"email_change_sent_at,omitempty"`

	LastSignInAt *time.Time `json:"last_sign_in_at,omitempty"`

	AppMetaData    map[string]interface{} `json:"app_metadata" sql:"-"`
	RawAppMetaData string                 `json:"-"`

	UserMetaData    map[string]interface{} `json:"user_metadata" sql:"-"`
	RawUserMetaData string                 `json:"-"`

	IsSuperAdmin bool `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewUser initializes a new user from an email, password and user data.
func NewUser(instanceID string, email, password, aud string, userData map[string]interface{}) (*User, error) {
	user := &User{
		InstanceID:   instanceID,
		ID:           uuid.NewRandom().String(),
		Aud:          aud,
		Email:        email,
		UserMetaData: userData,
	}

	if err := user.EncryptPassword(password); err != nil {
		return nil, err
	}

	user.GenerateConfirmationToken()
	return user, nil
}

func NewSystemUser(instanceID, aud string) *User {
	return &User{
		InstanceID:   instanceID,
		ID:           SystemUserID,
		Aud:          aud,
		IsSuperAdmin: true,
	}
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	return u.BeforeUpdate()
}

func (u *User) AfterFind() (err error) {
	if u.RawAppMetaData != "" {
		err = json.Unmarshal([]byte(u.RawAppMetaData), &u.AppMetaData)
	} else {
		u.AppMetaData = make(map[string]interface{})
	}

	if err == nil && u.RawUserMetaData != "" {
		err = json.Unmarshal([]byte(u.RawUserMetaData), &u.UserMetaData)
	} else {
		u.UserMetaData = make(map[string]interface{})
	}

	return err
}

func (u *User) BeforeUpdate() error {
	if u.ID == SystemUserID {
		return errors.New("Cannot persist system user")
	}

	if u.AppMetaData != nil {
		data, err := json.Marshal(u.AppMetaData)
		if err != nil {
			return err
		}
		u.RawAppMetaData = string(data)
	}
	if u.UserMetaData != nil {
		data, err := json.Marshal(u.UserMetaData)
		if err != nil {
			return err
		}
		u.RawUserMetaData = string(data)
	}

	return nil
}

func (u *User) BeforeSave() error {
	if u.ID == SystemUserID {
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
func (u *User) SetRole(roleName string) {
	u.Role = strings.TrimSpace(roleName)
}

// HasRole returns true when the users role is set to roleName
func (u *User) HasRole(roleName string) bool {
	return u.Role == roleName
}

// UpdateUserMetaData sets all user data from a map of updates,
// ensuring that it doesn't override attributes that are not
// in the provided map.
func (u *User) UpdateUserMetaData(updates map[string]interface{}) {
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
}

// UpdateAppMetaData updates all app data from a map of updates
func (u *User) UpdateAppMetaData(updates map[string]interface{}) {
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
}

// EncryptPassword sets the encrypted password from a plaintext string
func (u *User) EncryptPassword(password string) error {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.EncryptedPassword = string(pw)
	return nil
}

// Authenticate a user from a password
func (u *User) Authenticate(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password))
	return err == nil
}

// GenerateConfirmationToken generates a secure confirmation token for confirming
// signup
func (u *User) GenerateConfirmationToken() {
	token := crypto.SecureToken()
	u.ConfirmationToken = token
}

// GenerateRecoveryToken generates a secure password recovery token
func (u *User) GenerateRecoveryToken() {
	token := crypto.SecureToken()
	now := time.Now()
	u.RecoveryToken = token
	u.RecoverySentAt = &now
}

// GenerateEmailChange prepares for verifying a new email
func (u *User) GenerateEmailChange(email string) {
	token := crypto.SecureToken()
	now := time.Now()
	u.EmailChangeToken = token
	u.EmailChangeSentAt = &now
	u.EmailChange = email
}

// Confirm resets the confimation token and the confirm timestamp
func (u *User) Confirm() {
	u.ConfirmationToken = ""
	now := time.Now()
	u.ConfirmedAt = &now
}

// ConfirmEmailChange confirm the change of email for a user
func (u *User) ConfirmEmailChange() {
	u.Email = u.EmailChange
	u.EmailChange = ""
	u.EmailChangeToken = ""
}

// Recover resets the recovery token
func (u *User) Recover() {
	u.RecoveryToken = ""
}

// TableName returns the namespaced user table name
func (*User) TableName() string {
	return tableName("users")
}
