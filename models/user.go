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

	Data []Data `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NUMBER|STRING|BOOL are the different types supported in custom data for users
const (
	NUMBER = iota
	STRING
	BOOL
)

// Data is the custom data on a user
type Data struct {
	UserID string `gorm:"primary_key"`
	Key    string `gorm:"primary_key"`

	Type int

	NumericValue float64
	StringValue  string
	BoolValue    bool
}

// Value returns the value of the data field
func (d *Data) Value() interface{} {
	switch d.Type {
	case STRING:
		return d.StringValue
	case NUMBER:
		return d.NumericValue
	case BOOL:
		return d.BoolValue
	}
	return nil
}

// InvalidDataType is an error returned when trying to set an invalid datatype for
// a user data key
type InvalidDataType struct {
	Key string
}

func (i *InvalidDataType) Error() string {
	return "Invalid datatype for data field " + i.Key + " only strings, numbers and bools allowed"
}

func userDataToMap(data []Data) map[string]interface{} {
	result := map[string]interface{}{}
	for _, field := range data {
		switch field.Type {
		case NUMBER:
			result[field.Key] = field.NumericValue
		case STRING:
			result[field.Key] = field.StringValue
		case BOOL:
			result[field.Key] = field.BoolValue
		}
	}
	return result
}

// MarshalJSON is a custom JSON marshaller for Users
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		Data map[string]interface{} `json:"data"`
	}{
		Alias: (*Alias)(u),
		Data:  userDataToMap(u.Data),
	})
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

// UpdateUserData updates all user data from a map of updates
func (u *User) UpdateUserData(tx *gorm.DB, updates *map[string]interface{}) error {
	for key, value := range *updates {
		data := &Data{}
		result := tx.First(data, "user_id = ? and key = ?", u.ID, key)
		data.UserID = u.ID
		data.Key = key
		if result.Error != nil && !result.RecordNotFound() {
			tx.Rollback()
			return result.Error
		}
		if value == nil {
			tx.Delete(data)
			continue
		}
		switch v := value.(type) {
		case string:
			data.StringValue = v
			data.Type = STRING
		case float64:
			data.NumericValue = v
			data.Type = NUMBER
		case bool:
			data.BoolValue = v
			data.Type = BOOL
		default:
			tx.Rollback()
			return &InvalidDataType{key}
		}
		if result.RecordNotFound() {
			tx.Create(data)
		} else {
			tx.Save(data)
		}
	}
	return nil
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
