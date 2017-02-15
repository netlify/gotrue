package sql

import (
	"encoding/json"

	"github.com/jinzhu/gorm"
	"github.com/netlify/netlify-auth/models"
	"github.com/pkg/errors"
)

type UserObj struct {
	FirstRoleName  string `json:"-" sql:"-"`
	AutoAsignRoles bool   `json:"-" sql:"-"`

	*models.User

	RawAppMetaData  string `json:"-" bson:"-"`
	RawUserMetaData string `json:"-" bson:"-"`
}

func (u *UserObj) BeforeCreate(tx *gorm.DB) error {
	if !u.AutoAsignRoles {
		return nil
	}

	var userCount int64
	if result := tx.Table(u.TableName()).Where("id != ?", u.ID).Count(&userCount); result.Error != nil {
		return errors.Wrap(result.Error, "error finding registered users")
	}

	if userCount == 0 {
		u.SetRole(u.FirstRoleName)
	}

	return u.BeforeUpdate()
}

func (u *UserObj) AfterFind() (err error) {
	if u.RawAppMetaData != "" {
		err = json.Unmarshal([]byte(u.RawAppMetaData), &u.AppMetaData)
	}

	if err == nil && u.RawUserMetaData != "" {
		err = json.Unmarshal([]byte(u.RawUserMetaData), &u.UserMetaData)
	}

	return err
}

func (u *UserObj) BeforeUpdate() (err error) {
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
