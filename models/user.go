package models

import (
	"time"

	"github.com/jinzhu/gorm"

	"golang.org/x/crypto/bcrypt"
)

// User respresents a registered user with email/password authentication
type User struct {
	ID int64 `json:"id"`

	Email              string    `json:"email"`
	EncryptedPassword  string    `json:"-"`
	ConfirmedAt        time.Time `json:"confirmed_at"`
	ConfirmationToken  string    `json:"-"`
	ConfirmationSentAt time.Time `json:"confirmation_sent_at,omitempty"`
	LastSignInAt       time.Time `json:"last_sign_in_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUser creates a new user from an email and password
func CreateUser(db *gorm.DB, email, password string) (*User, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		Email:             email,
		EncryptedPassword: string(pw),
	}

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
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

// Confirm resets the confimation token and the confirm timestamp
func (u *User) Confirm() {
	u.ConfirmationToken = ""
	u.ConfirmedAt = time.Now()
}
