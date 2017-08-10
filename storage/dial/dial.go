package dial

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/mongo"
	"github.com/netlify/gotrue/storage/sql"
)

// Dial will connect to that storage engine
func Dial(config *conf.GlobalConfiguration) (storage.Connection, error) {
	if config.DB.Namespace != "" {
		models.Namespace = config.DB.Namespace
	}

	if config.DB.Driver == "mongo" {
		return mongo.Dial(config)
	}

	return sql.Dial(config)
}
