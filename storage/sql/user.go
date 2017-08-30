package sql

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
	"github.com/netlify/gotrue/models"
)

type userObj struct {
	*models.User

	RawAppMetaData  string `json:"-"`
	RawUserMetaData string `json:"-"`
}

func (u *userObj) BeforeCreate(tx *gorm.DB) error {
	return u.BeforeUpdate()
}

func (u *userObj) AfterFind() (err error) {
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

func (u *userObj) BeforeUpdate() error {
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

func (u *userObj) BeforeSave() error {
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

func (conn *Connection) newUserObj(user *models.User) *userObj {
	return &userObj{
		User: user,
	}
}
