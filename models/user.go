package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/netlify/netlify-auth/crypto"
	"github.com/pborman/uuid"

	"golang.org/x/crypto/bcrypt"
)

type VerifyType string

const (
	ConfirmationVerifyType VerifyType = "confirmation_token"
	RecoveryVerifyType     VerifyType = "recovery_token"
)

// User respresents a registered user with email/password authentication
type User struct {
	ID string `json:"id"`

	Email             string    `json:"email"`
	EncryptedPassword string    `json:"-"`
	ConfirmedAt       time.Time `json:"confirmed_at"`

	ConfirmationToken  string    `json:"-"`
	ConfirmationSentAt time.Time `json:"confirmation_sent_at,omitempty"`

	RecoveryToken  string    `json:"-"`
	RecoverySentAt time.Time `json:"recovery_sent_at,omitempty"`

	EmailChangeToken  string    `json:"-"`
	EmailChange       string    `json:"new_email,ommitempty"`
	EmailChangeSentAt time.Time `json:"email_change_sent_at,omitempty"`

	LastSignInAt time.Time `json:"last_sign_in_at,omitempty"`

	AppMetaData     map[string]interface{} `json:"app_metadata,omitempty" sql:"-"`
	UserMetaData    map[string]interface{} `json:"user_metadata,omitempty" sql:"-"`
	RawAppMetaData  string                 `json:"-"`
	RawUserMetaData string                 `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return tableName("users")
}

func (u *User) AfterFind() (err error) {
	if u.RawAppMetaData != "" {
		err = json.Unmarshal([]byte(u.RawAppMetaData), &u.AppMetaData)
	}
	if err == nil && u.RawUserMetaData != "" {
		err = json.Unmarshal([]byte(u.RawUserMetaData), &u.UserMetaData)
	}
	return err
}

func (u *User) BeforeUpdate() (err error) {
	if u.AppMetaData != nil {
		data, err := json.Marshal(u.AppMetaData)
		if err == nil {
			u.RawAppMetaData = string(data)
		}
	}
	if u.UserMetaData != nil {
		data, err := json.Marshal(u.UserMetaData)
		if err == nil {
			u.RawUserMetaData = string(data)
		}
	}
	return err
}

// NewUser initializes a new user from an email, password and user data.
func NewUser(email, password string, userData map[string]interface{}) (*User, error) {
	user := &User{
		ID:           uuid.NewRandom().String(),
		Email:        email,
		UserMetaData: userData,
	}

	if err := user.EncryptPassword(password); err != nil {
		return nil, err
	}

	user.GenerateConfirmationToken()
	return user, nil
}

// IsRegistered checks if a user has already being
// registered and confirmed.
func (u *User) IsRegistered() bool {
	return !u.ConfirmedAt.IsZero()
}

func (u *User) SetRole(roleName string) {
	newRole := strings.TrimSpace(roleName)

	if u.AppMetaData == nil {
		u.AppMetaData = map[string]interface{}{"roles": []string{newRole}}
	} else if roles, ok := u.AppMetaData["roles"]; ok {
		if rolesSlice, ok := roles.([]string); ok {
			for _, role := range rolesSlice {
				if role == newRole {
					return
				}
			}
			u.AppMetaData["roles"] = append(rolesSlice, newRole)
		}
	} else {
		u.AppMetaData["roles"] = []string{newRole}
	}
}

// HasRole checks if app_metadata.roles includes the specified role
func (u *User) HasRole(role string) bool {
	if u.AppMetaData == nil {
		return false
	}
	roles, ok := u.AppMetaData["roles"]
	if !ok {
		return false
	}
	roleStrings, ok := roles.([]string)
	if !ok {
		return false
	}
	for _, r := range roleStrings {
		if r == role {
			return true
		}
	}
	return false
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
	u.ConfirmationSentAt = time.Now()
}

// GenerateRecoveryToken generates a secure password recovery token
func (u *User) GenerateRecoveryToken() {
	token := crypto.SecureToken()
	u.RecoveryToken = token
	u.RecoverySentAt = time.Now()
}

// GenerateEmailChangeToken prepares for verifying a new email
func (u *User) GenerateEmailChange(email string) {
	token := crypto.SecureToken()
	u.EmailChangeToken = token
	u.EmailChangeSentAt = time.Now()
	u.EmailChange = email
}

// Confirm resets the confimation token and the confirm timestamp
func (u *User) Confirm() {
	u.ConfirmationToken = ""
	u.ConfirmedAt = time.Now()
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
