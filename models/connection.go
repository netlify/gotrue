package models

import (
	// this is where we do the connections
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/netlify/netlify-auth/conf"
	"github.com/pkg/errors"

	"github.com/jinzhu/gorm"
)

// Namespace puts all tables names under a common
// namespace. This is useful if you want to use
// the same database for several services and don't
// want table names to collide.
var Namespace string

// Configuration defines the info necessary to connect to a storage engine
type Configuration struct {
	Driver  string `json:"driver"`
	ConnURL string `json:"conn_url"`
}

// Connect will connect to that storage engine
func Connect(config *conf.Configuration) (*gorm.DB, error) {
	db, err := gorm.Open(config.DB.Driver, config.DB.ConnURL)
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}

	err = db.DB().Ping()
	if err != nil {
		return nil, errors.Wrap(err, "checking database connection")
	}

	if config.DB.Automigrate {
		if err := AutoMigrate(db); err != nil {
			return nil, errors.Wrap(err, "migrating tables")
		}
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	db = db.AutoMigrate(User{}, RefreshToken{}, UserData{})
	return db.Error
}

func tableName(defaultName string) string {
	if Namespace != "" {
		return Namespace + "_" + defaultName
	}
	return defaultName
}
