package sql

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
	"github.com/netlify/gotrue/models"
)

type userObj struct {
	*models.User

	RawAppMetaData  string `json:"-" bson:"-"`
	RawUserMetaData string `json:"-" bson:"-"`
}

func (u *userObj) BeforeCreate(tx *gorm.DB) error {
	return u.BeforeUpdate()
}

func (u *userObj) AfterFind() (err error) {
	if u.RawAppMetaData != "" {
		err = json.Unmarshal([]byte(u.RawAppMetaData), &u.AppMetaData)
	}

	if err == nil && u.RawUserMetaData != "" {
		err = json.Unmarshal([]byte(u.RawUserMetaData), &u.UserMetaData)
	}

	return err
}

func (u *userObj) BeforeUpdate() (err error) {
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

func (conn *Connection) newUserObj(user *models.User) *userObj {
	return &userObj{
		User: user,
	}
}
