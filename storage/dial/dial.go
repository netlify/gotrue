package dial

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/sql"
)

// Dial will connect to that storage engine
func Dial(config *conf.GlobalConfiguration) (storage.Connection, error) {
	conn, err := sql.Dial(config)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
