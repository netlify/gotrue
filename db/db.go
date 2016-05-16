package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // Import SQLite
)

// DB is a wrapper around our Database
type DB struct {
	db *gorm.DB
}

// NewDB returns a new database
func NewDB() (*DB, error) {
	db, err := gorm.Open("sqlite3", "gorm.db")
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, err
}
