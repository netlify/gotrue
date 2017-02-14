package sql

import (
	"github.com/jinzhu/gorm"
	"github.com/netlify/netlify-auth/models"
	"github.com/pkg/errors"
)

type UserObj struct {
	FirstRoleName  string `json:"-" sql:"-"`
	AutoAsignRoles bool   `json:"-" sql:"-"`
	*models.User
}

func (u *UserObj) BeforeCreate(tx *gorm.DB) error {
	if !u.AutoAsignRoles {
		return nil
	}

	var userCount int64
	if result := tx.Where("id != ?", u.ID).Count(&userCount); result.Error != nil {
		return errors.Wrap(result.Error, "error finding registered users")
	}

	if userCount == 0 {
		u.User.SetRole(u.FirstRoleName)
	}
	return nil
}
