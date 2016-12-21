package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"

	"golang.org/x/crypto/bcrypt"
)

// NUMBER|STRING|BOOL are the different types supported in custom data for users
const (
	NUMBER = iota
	STRING
	BOOL
	MAP
	ARRAY
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

	AppMetaData  []UserData `gorm:"polymorphic:User;polymorphic_value:app;" json:"-"`
	UserMetaData []UserData `gorm:"polymorphic:User;polymorphic_value:user" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return tableName("users")
}

// UserData is the custom data on a user
type UserData struct {
	UserID   string `gorm:"primary_key"`
	UserType string `gorm:"primary_key"`
	Key      string `gorm:"primary_key"`

	Type int

	NumericValue    float64
	StringValue     string
	BoolValue       bool
	SerializedValue string
}

func (UserData) TableName() string {
	return tableName("users_data")
}

// Value returns the value of the data field
func (d *UserData) Value() interface{} {
	switch d.Type {
	case STRING:
		return d.StringValue
	case NUMBER:
		return d.NumericValue
	case BOOL:
		return d.BoolValue
	case MAP:
		data := &map[string]interface{}{}
		json.Unmarshal([]byte(d.SerializedValue), data)
		return data
	case ARRAY:
		data := &[]interface{}{}
		err := json.Unmarshal([]byte(d.SerializedValue), data)
		if err != nil {
			fmt.Printf("Error unserializing %v: %v\n", d.Key, err)
		}
		return data
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

// NewUserData returns a new userdata from a key and value
func NewUserData(key string, value interface{}) (UserData, error) {
	data := UserData{Key: key}
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
		if reflect.TypeOf(value).Kind() == reflect.Slice {
			data.Type = ARRAY
		} else {
			data.Type = MAP
		}
		serialized, error := json.Marshal(value)
		if error != nil {
			return data, error
		}
		data.SerializedValue = string(serialized[:])
	}
	return data, nil
}

func userDataToMap(data []UserData) map[string]interface{} {
	result := map[string]interface{}{}
	for _, field := range data {
		result[field.Key] = field.Value()
	}
	return result
}

// MarshalJSON is a custom JSON marshaller for Users
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		AppMetaData  map[string]interface{} `json:"app_metadata"`
		UserMetaData map[string]interface{} `json:"user_metadata"`
	}{
		Alias:        (*Alias)(u),
		AppMetaData:  u.AppMetaDataMap(),
		UserMetaData: u.UserMetaDataMap(),
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

// AddRole adds a role in the app metadata
func (u *User) AddRole(tx *gorm.DB, role string) error {
	roles := []string{}
	existing, ok := userDataToMap(u.AppMetaData)["roles"]
	if ok {
		roles, _ = existing.([]string)
	}
	for _, r := range roles {
		if r == role {
			return nil
		}
	}
	roles = append(roles, role)
	updates := &map[string]interface{}{"roles": roles}
	return u.UpdateAppMetaData(tx, updates)
}

// RemoveRole remoes a role in the app metadata
func (u *User) RemoveRole(tx *gorm.DB, role string) error {
	roles := []string{}
	existing, ok := userDataToMap(u.AppMetaData)["roles"]
	if ok {
		roles, _ = existing.([]string)
	}
	newRoles := []string{}
	for _, r := range roles {
		if r != role {
			newRoles = append(newRoles, r)
		}
	}
	updates := &map[string]interface{}{"roles": newRoles}
	return u.UpdateAppMetaData(tx, updates)
}

// HasRole checks if app_metadata.roles includes the specified role
func (u *User) HasRole(role string) bool {
	for _, data := range u.AppMetaData {
		if data.Key == "roles" && data.Type == ARRAY {
			roles := []string{}
			err := json.Unmarshal([]byte(data.SerializedValue), &roles)
			if err != nil {
				return false
			}
			for _, r := range roles {
				if r == role {
					return true
				}
			}
		}
	}
	return false
}

func updateUserData(userData []UserData, updates *map[string]interface{}) ([]UserData, error) {
	existing := userDataToMap(userData)
	for key, value := range *updates {
		if value == nil {
			delete(existing, key)
		} else {
			existing[key] = value
		}
	}

	newUserData := make([]UserData, len(existing))
	i := 0
	for key, value := range existing {
		data, err := NewUserData(key, value)
		if err != nil {
			return nil, err
		}
		newUserData[i] = data
		i++
	}
	return newUserData, nil
}

// UpdateUserData updates all user data from a map of updates
func (u *User) UpdateUserMetaData(tx *gorm.DB, updates *map[string]interface{}) error {
	userMetaData, err := updateUserData(u.UserMetaData, updates)
	if err != nil {
		return err
	}
	tx.Model(u).Association("UserMetaData").Replace(userMetaData)
	return nil
}

// UpdateUserData updates all user data from a map of updates
func (u *User) UpdateAppMetaData(tx *gorm.DB, updates *map[string]interface{}) error {
	appMetaData, err := updateUserData(u.AppMetaData, updates)
	if err != nil {
		return err
	}
	tx.Model(u).Association("AppMetaData").Replace(appMetaData)
	return nil
}

func (u *User) AppMetaDataMap() map[string]interface{} {
	return userDataToMap(u.AppMetaData)
}

func (u *User) UserMetaDataMap() map[string]interface{} {
	return userDataToMap(u.UserMetaData)
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
