package storage

import (
	"gorm.io/gorm"
)

var migrations []interface{}

func AddMigration(obj interface{}) {
	migrations = append(migrations, obj)
}

func MigrateDatabase(conn *Connection) error {
	// AddMigration the schema
	return conn.AutoMigrate(migrations...)
}

func TruncateAll(conn *Connection) error {
	return conn.Transaction(func(tx *Connection) error {
		for _, m := range migrations {
			stmt := &gorm.Statement{DB: conn.DB}
			if err := stmt.Parse(m); err != nil {
				return err
			}
			if err := tx.Exec("TRUNCATE TABLE " + stmt.Schema.Table).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
