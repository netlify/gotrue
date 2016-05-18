package models

import (
	// this is where we do the connections
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/netlify/authlify/conf"

	"github.com/jinzhu/gorm"
)

// Configuration defines the info necessary to connect to a storage engine
type Configuration struct {
	Driver  string `json:"driver"`
	ConnURL string `json:"conn_url"`
}

// Connect will connect to that storage engine
func Connect(config *conf.Configuration) (*gorm.DB, error) {
	db, err := gorm.Open(config.DB.Driver, config.DB.ConnURL)
	if err != nil {
		return nil, err
	}

	err = db.DB().Ping()
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&User{}, &RefreshToken{}, &Data{})

	return db, nil
}
