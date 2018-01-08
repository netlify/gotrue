package test

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/dial"
)

func SetupDBConnection(globalConfig *conf.GlobalConfiguration) (storage.Connection, error) {
	var conn storage.Connection
	conn, err := dial.Dial(globalConfig)
	if err != nil {
		return nil, err
	}

	// if globalConfig.DB.Automigrate {
	// 	if err = conn.MigrateUp(); err != nil {
	// 		conn.Close()
	// 		return nil, err
	// 	}
	// }
	return conn, err
}

// func CreateTestDB(dbname string, config *conf.GlobalConfiguration) (string, error) {
// 	dburl := fmt.Sprintf("root@tcp(127.0.0.1:3306)/%s?parseTime=true&multiStatements=true", dbname)
// 	db, err := pop.NewConnection(&pop.ConnectionDetails{
// 		Dialect: config.DB.Driver,
// 		URL:     dburl,
// 	})
// 	if err != nil {
// 		return "", err
// 	}
// 	return dburl, pop.CreateDB(db)
// }

// func DeleteTestDB(config *conf.GlobalConfiguration) error {
// 	db, err := pop.NewConnection(&pop.ConnectionDetails{
// 		Dialect: config.DB.Driver,
// 		URL:     config.DB.URL,
// 	})
// 	if err != nil {
// 		return err
// 	}
// 	return pop.DropDB(db)
// }
