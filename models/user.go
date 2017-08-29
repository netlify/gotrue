package models

import (
	"strings"
	"time"

	"github.com/netlify/gotrue/crypto"
	"github.com/pborman/uuid"

	"golang.org/x/crypto/bcrypt"
)

// User respresents a registered user with email/password authentication
type User struct {
	InstanceID string `json:"-" bson:"instance_id"`
	ID         string `json:"id" bson:"_id,omitempty"`

	Aud               string     `json:"aud" bson:"aud"`
	Role              string     `json:"role" bson:"role"`
	Email             string     `json:"email" bson:"email"`
	EncryptedPassword string     `json:"-" bson:"encrypted_password"`
	ConfirmedAt       *time.Time `json:"confirmed_at" bson:"confirmed_at"`
	InvitedAt         *time.Time `json:"invited_at" bson:"invited_at"`

	ConfirmationToken  string     `json:"-" bson:"confirmation_token,omitempty"`
	ConfirmationSentAt *time.Time `json:"confirmation_sent_at,omitempty" bson:"confirmation_sent_at,omitempty"`

	RecoveryToken  string     `json:"-" bson:"recovery_token,omitempty"`
	RecoverySentAt *time.Time `json:"recovery_sent_at,omitempty" bson:"recovery_sent_at,omitempty"`

	EmailChangeToken  string     `json:"-" bson:"email_change_token,omitempty"`
	EmailChange       string     `json:"new_email,omitempty" bson:"new_email,omitempty"`
	EmailChangeSentAt *time.Time `json:"email_change_sent_at,omitempty" bson:"email_change_sent_at,omitempty"`

	LastSignInAt *time.Time `json:"last_sign_in_at,omitempty" bson:"last_sign_in_at,omitempty"`

	AppMetaData  map[string]interface{} `json:"app_metadata,omitempty" sql:"-" bson:"app_metadata,omitempty"`
	UserMetaData map[string]interface{} `json:"user_metadata,omitempty" sql:"-" bson:"user_metadata,omitempty"`

	IsSuperAdmin bool `json:"-" bson:"is_super_admin"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
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
			u.UserMetaData[key] = value
		}
	}
}

// UpdateAppMetaData updates all app data from a map of updates
func (u *User) UpdateAppMetaData(updates map[string]interface{}) {
	if u.AppMetaData == nil {
		u.AppMetaData = updates
	} else if updates != nil {
		for key, value := range updates {
			u.AppMetaData[key] = value
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
