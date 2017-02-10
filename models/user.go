package models

import (
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"

	"golang.org/x/crypto/bcrypt"
)

// User respresents a registered user with email/password authentication
type User struct {
	ID string `json:"id"`

	Email              string    `json:"email"`
	EncryptedPassword  string    `json:"-"`
	ConfirmedAt        time.Time `json:"confirmed_at"`
	ConfirmationToken  string    `json:"-"`
	ConfirmationSentAt time.Time `json:"confirmation_sent_at,omitempty"`
	RecoveryToken      string    `json:"-"`
	RecoverySentAt     time.Time `json:"recovery_sent_at,omitempty"`
	LastSignInAt       time.Time `json:"last_sign_in_at,omitempty"`

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

func (u *User) AfterFind() error {
	if u.RawAppMetaData != "" {
		return json.Unmarshal([]byte(u.RawAppMetaData), &u.AppMetaData)
	}
	if u.RawUserMetaData != "" {
		return json.Unmarshal([]byte(u.RawUserMetaData), &u.UserMetaData)
	}
	return nil
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

// CreateUser creates a new user from an email and password
func CreateUser(db *gorm.DB, email, password string) (*User, error) {
	user := &User{
		ID:    uuid.NewRandom().String(),
		Email: email,
	}

	if err := user.EncryptPassword(password); err != nil {
		return nil, err
	}

	user.GenerateConfirmationToken()

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
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

// UpdateUserData updates all user data from a map of updates
func (u *User) UpdateUserMetaData(tx *gorm.DB, updates *map[string]interface{}) error {
	if u.UserMetaData == nil {
		u.UserMetaData = *updates
	} else {
		for key, value := range *updates {
			u.UserMetaData[key] = value
		}
	}
	return tx.Save(u).Error
}

// UpdateUserData updates all user data from a map of updates
func (u *User) UpdateAppMetaData(tx *gorm.DB, updates *map[string]interface{}) error {
	if u.AppMetaData == nil {
		u.AppMetaData = *updates
	} else {
		for key, value := range *updates {
			u.AppMetaData[key] = value
		}
	}
	return tx.Save(u).Error
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
	token := secureToken()
	u.ConfirmationToken = token
	u.ConfirmationSentAt = time.Now()
}

// GenerateRecoveryToken generates a secure password recovery token
func (u *User) GenerateRecoveryToken() {
	token := secureToken()
	u.RecoveryToken = token
	u.RecoverySentAt = time.Now()
}

// Confirm resets the confimation token and the confirm timestamp
func (u *User) Confirm() {
	u.ConfirmationToken = ""
	u.ConfirmedAt = time.Now()
}

// Recover resets the recovery token
func (u *User) Recover() {
	u.RecoveryToken = ""
}
