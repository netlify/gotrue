package models

import (
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

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUser creates a new user from an email and password
func CreateUser(db *gorm.DB, email, password string) (*User, error) {
	user := &User{
		ID:    uuid.NewRandom().String(),
		Email: email,
	}

	if err := user.UpdatePassword(password); err != nil {
		return nil, err
	}

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// UpdatePassword sets the encrypted password from a plaintext string
func (u *User) UpdatePassword(password string) error {
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
