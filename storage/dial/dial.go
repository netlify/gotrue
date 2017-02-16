package dial

import (
	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/models"
	"github.com/netlify/netlify-auth/storage"
	"github.com/netlify/netlify-auth/storage/mongo"
	"github.com/netlify/netlify-auth/storage/sql"
)

// Dial will connect to that storage engine
func Dial(config *conf.Configuration) (storage.Connection, error) {
	if config.DB.Namespace != "" {
		models.Namespace = config.DB.Namespace
	}

	if config.DB.Driver == "mongo" {
		return mongo.Dial(config)
	}

	return sql.Dial(config)
}
