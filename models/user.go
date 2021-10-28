package models

import (
	"database/sql"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/storage"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const SystemUserID = "0"

var SystemUserUUID = uuid.Nil

// User respresents a registered user with email/password authentication
type User struct {
	InstanceID uuid.UUID `json:"-" db:"instance_id"`
	ID         uuid.UUID `json:"id" db:"id"`

	Aud               string             `json:"aud" db:"aud"`
	Role              string             `json:"role" db:"role"`
	Email             storage.NullString `json:"email" db:"email"`
	EncryptedPassword string             `json:"-" db:"encrypted_password"`
	EmailConfirmedAt  *time.Time         `json:"email_confirmed_at,omitempty" db:"email_confirmed_at"`
	InvitedAt         *time.Time         `json:"invited_at,omitempty" db:"invited_at"`

	Phone            storage.NullString `json:"phone" db:"phone"`
	PhoneConfirmedAt *time.Time         `json:"phone_confirmed_at,omitempty" db:"phone_confirmed_at"`

	ConfirmationToken  string     `json:"-" db:"confirmation_token"`
	ConfirmationSentAt *time.Time `json:"confirmation_sent_at,omitempty" db:"confirmation_sent_at"`

	// For backward compatibility only. Use EmailConfirmedAt or PhoneConfirmedAt instead.
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty" db:"confirmed_at" rw:"r"`

	RecoveryToken  string     `json:"-" db:"recovery_token"`
	RecoverySentAt *time.Time `json:"recovery_sent_at,omitempty" db:"recovery_sent_at"`

	EmailChangeTokenCurrent  string     `json:"-" db:"email_change_token_current"`
	EmailChangeTokenNew      string     `json:"-" db:"email_change_token_new"`
	EmailChange              string     `json:"new_email,omitempty" db:"email_change"`
	EmailChangeSentAt        *time.Time `json:"email_change_sent_at,omitempty" db:"email_change_sent_at"`
	EmailChangeConfirmStatus int        `json:"-" db:"email_change_confirm_status"`

	PhoneChangeToken  string     `json:"-" db:"phone_change_token"`
	PhoneChange       string     `json:"new_phone,omitempty" db:"phone_change"`
	PhoneChangeSentAt *time.Time `json:"phone_change_sent_at,omitempty" db:"phone_change_sent_at"`

	LastSignInAt *time.Time `json:"last_sign_in_at,omitempty" db:"last_sign_in_at"`

	AppMetaData  JSONMap `json:"app_metadata" db:"raw_app_meta_data"`
	UserMetaData JSONMap `json:"user_metadata" db:"raw_user_meta_data"`

	IsSuperAdmin bool       `json:"-" db:"is_super_admin"`
	Identities   []Identity `json:"identities" has_many:"identities"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// NewUser initializes a new user from an email, password and user data.
// TODO: Refactor NewUser to take in phone as an arg
func NewUser(instanceID uuid.UUID, email, password, aud string, userData map[string]interface{}) (*User, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, errors.Wrap(err, "Error generating unique id")
	}
	pw, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	if userData == nil {
		userData = make(map[string]interface{})
	}
	user := &User{
		InstanceID:        instanceID,
		ID:                id,
		Aud:               aud,
		Email:             storage.NullString(strings.ToLower(email)),
		UserMetaData:      userData,
		EncryptedPassword: pw,
	}
	return user, nil
}

// NewSystemUser returns a user with the id as SystemUserUUID
func NewSystemUser(instanceID uuid.UUID, aud string) *User {
	return &User{
		InstanceID:   instanceID,
		ID:           SystemUserUUID,
		Aud:          aud,
		IsSuperAdmin: true,
	}
}

// TableName overrides the table name used by pop
func (User) TableName() string {
	tableName := "users"
	return tableName
}

// BeforeCreate is invoked before a create operation is ran
func (u *User) BeforeCreate(tx *pop.Connection) error {
	return u.BeforeUpdate(tx)
}

// BeforeUpdate is invoked before an update operation is ran
func (u *User) BeforeUpdate(tx *pop.Connection) error {
	if u.ID == SystemUserUUID {
		return errors.New("Cannot persist system user")
	}

	return nil
}

// BeforeSave is invoked before the user is saved to the database
func (u *User) BeforeSave(tx *pop.Connection) error {
	if u.ID == SystemUserUUID {
		return errors.New("Cannot persist system user")
	}

	if u.EmailConfirmedAt != nil && u.EmailConfirmedAt.IsZero() {
		u.EmailConfirmedAt = nil
	}
	if u.PhoneConfirmedAt != nil && u.PhoneConfirmedAt.IsZero() {
		u.PhoneConfirmedAt = nil
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
	if u.PhoneChangeSentAt != nil && u.PhoneChangeSentAt.IsZero() {
		u.PhoneChangeSentAt = nil
	}
	if u.LastSignInAt != nil && u.LastSignInAt.IsZero() {
		u.LastSignInAt = nil
	}
	return nil
}

// IsConfirmed checks if a user has already been
// registered and confirmed.
func (u *User) IsConfirmed() bool {
	return u.EmailConfirmedAt != nil
}

// IsPhoneConfirmed checks if a user's phone has already been
// registered and confirmed.
func (u *User) IsPhoneConfirmed() bool {
	return u.PhoneConfirmedAt != nil
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

// GetEmail returns the user's email as a string
func (u *User) GetEmail() string {
	return string(u.Email)
}

// GetPhone returns the user's phone number as a string
func (u *User) GetPhone() string {
	return string(u.Phone)
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

// UpdateAppMetaDataProviders updates the provider field in AppMetaData column
func (u *User) UpdateAppMetaDataProviders(tx *storage.Connection) error {
	providers, terr := FindProvidersByUser(tx, u)
	if terr != nil {
		return terr
	}
	return u.UpdateAppMetaData(tx, map[string]interface{}{
		"providers": providers,
	})
}

// SetEmail sets the user's email
func (u *User) SetEmail(tx *storage.Connection, email string) error {
	u.Email = storage.NullString(email)
	return tx.UpdateOnly(u, "email")
}

// SetPhone sets the user's phone
func (u *User) SetPhone(tx *storage.Connection, phone string) error {
	u.Phone = storage.NullString(phone)
	return tx.UpdateOnly(u, "phone")
}

// hashPassword generates a hashed password from a plaintext string
func hashPassword(password string) (string, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// UpdatePassword updates the user's password
func (u *User) UpdatePassword(tx *storage.Connection, password string) error {
	pw, err := hashPassword(password)
	if err != nil {
		return err
	}
	u.EncryptedPassword = pw
	return tx.UpdateOnly(u, "encrypted_password")
}

// UpdatePhone updates the user's phone
func (u *User) UpdatePhone(tx *storage.Connection, phone string) error {
	u.Phone = storage.NullString(phone)
	return tx.UpdateOnly(u, "phone")
}

// Authenticate a user from a password
func (u *User) Authenticate(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password))
	return err == nil
}

// Confirm resets the confimation token and sets the confirm timestamp
func (u *User) Confirm(tx *storage.Connection) error {
	u.ConfirmationToken = ""
	now := time.Now()
	u.EmailConfirmedAt = &now
	return tx.UpdateOnly(u, "confirmation_token", "email_confirmed_at")
}

// ConfirmPhone resets the confimation token and sets the confirm timestamp
func (u *User) ConfirmPhone(tx *storage.Connection) error {
	u.ConfirmationToken = ""
	now := time.Now()
	u.PhoneConfirmedAt = &now
	return tx.UpdateOnly(u, "confirmation_token", "phone_confirmed_at")
}

// UpdateLastSignInAt update field last_sign_in_at for user according to specified field
func (u *User) UpdateLastSignInAt(tx *storage.Connection) error {
	return tx.UpdateOnly(u, "last_sign_in_at")
}

// ConfirmEmailChange confirm the change of email for a user
func (u *User) ConfirmEmailChange(tx *storage.Connection, status int) error {
	u.Email = storage.NullString(u.EmailChange)
	u.EmailChange = ""
	u.EmailChangeTokenCurrent = ""
	u.EmailChangeTokenNew = ""
	u.EmailChangeConfirmStatus = status
	return tx.UpdateOnly(
		u,
		"email",
		"email_change",
		"email_change_token_current",
		"email_change_token_new",
		"email_change_confirm_status",
	)
}

// ConfirmPhoneChange confirms the change of phone for a user
func (u *User) ConfirmPhoneChange(tx *storage.Connection) error {
	u.Phone = storage.NullString(u.PhoneChange)
	u.PhoneChange = ""
	u.PhoneChangeToken = ""
	now := time.Now()
	u.PhoneConfirmedAt = &now
	return tx.UpdateOnly(u, "phone", "phone_change", "phone_change_token", "phone_confirmed_at")
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
	if err := tx.Eager().Q().Where(query, args...).First(obj); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, UserNotFoundError{}
		}
		return nil, errors.Wrap(err, "error finding user")
	}

	return obj, nil
}

// FindUserByConfirmationToken finds users with the matching confirmation token.
func FindUserByConfirmationToken(tx *storage.Connection, token string) (*User, error) {
	user, err := findUser(tx, "confirmation_token = ?", token)
	if err != nil {
		return nil, ConfirmationTokenNotFoundError{}
	}
	return user, nil
}

// FindUserByEmailAndAudience finds a user with the matching email and audience.
func FindUserByEmailAndAudience(tx *storage.Connection, instanceID uuid.UUID, email, aud string) (*User, error) {
	return findUser(tx, "instance_id = ? and LOWER(email) = ? and aud = ?", instanceID, strings.ToLower(email), aud)
}

// FindUserByPhoneAndAudience finds a user with the matching email and audience.
func FindUserByPhoneAndAudience(tx *storage.Connection, instanceID uuid.UUID, phone, aud string) (*User, error) {
	return findUser(tx, "instance_id = ? and phone = ? and aud = ?", instanceID, phone, aud)
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

// FindUserByEmailChangeToken finds a user with the matching email change token.
func FindUserByEmailChangeToken(tx *storage.Connection, token string) (*User, error) {
	return findUser(tx, "email_change_token_current = ? or email_change_token_new = ?", token, token)
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
		q = q.Where("(email LIKE ? OR raw_user_meta_data->>'full_name' ILIKE ?)", lf, lf)
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

// FindUserWithPhoneAndPhoneChangeToken finds a user with the matching phone and phone change token
func FindUserWithPhoneAndPhoneChangeToken(tx *storage.Connection, phone, token string) (*User, error) {
	return findUser(tx, "phone = ? and phone_change_token = ?", phone, token)
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

// IsDuplicatedPhone checks if the phone number already exists in the users table
func IsDuplicatedPhone(tx *storage.Connection, instanceID uuid.UUID, phone, aud string) (bool, error) {
	_, err := FindUserByPhoneAndAudience(tx, instanceID, phone, aud)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
