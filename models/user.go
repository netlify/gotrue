package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

// User respresents a registered user with email/password authentication
type User struct {
	gorm.Model

	Email              string    `json:"email"`
	EncryptedPassword  string    `json:"-"`
	ConfirmedAt        time.Time `json:"confirmed_at"`
	ConfirmationToken  string    `json:"-"`
	ConfirmationSentAt time.Time `json:"confirmation_sent_at"`
	LastSignInAt       time.Time `json:"last_sign_in_at"`
}

// GenerateConfirmationToken generates a secure confirmation token for confirming
// signup
func (u *User) GenerateConfirmationToken() error {
	token, err := secureToken(32)
	if err != nil {
		return err
	}
	u.ConfirmationToken = token
	u.ConfirmationSentAt = time.Now()
	return nil
}
