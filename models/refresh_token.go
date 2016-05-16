package models

import "github.com/jinzhu/gorm"

type RefreshToken struct {
	gorm.Model

	User   User
	UserID int
}
