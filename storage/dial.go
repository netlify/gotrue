package storage

import (
	"net/url"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/pop/v5/columns"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage/namespace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Connection is the interface a storage provider must implement.
type Connection struct {
	*pop.Connection
}

// Dial will connect to that storage engine
func Dial(config *conf.GlobalConfiguration) (*Connection, error) {
	if config.DB.Driver == "" && config.DB.URL != "" {
		u, err := url.Parse(config.DB.URL)
		if err != nil {
			return nil, errors.Wrap(err, "parsing db connection url")
		}
		config.DB.Driver = u.Scheme
	}

	db, err := pop.NewConnection(&pop.ConnectionDetails{
		Dialect: config.DB.Driver,
		URL:     config.DB.URL,
	})
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}
	if err := db.Open(); err != nil {
		return nil, errors.Wrap(err, "checking database connection")
	}

	if config.DB.Namespace != "" {
		namespace.SetNamespace(config.DB.Namespace)
	}

	if logrus.StandardLogger().Level == logrus.DebugLevel {
		pop.Debug = true
	}

	return &Connection{db}, nil
}

func (c *Connection) Transaction(fn func(*Connection) error) error {
	if c.TX == nil {
		return c.Connection.Transaction(func(tx *pop.Connection) error {
			return fn(&Connection{tx})
		})
	}
	return fn(c)
}

func getExcludedColumns(model interface{}, includeColumns ...string) ([]string, error) {
	sm := &pop.Model{Value: model}

	// get all columns and remove included to get excluded set
	cols := columns.ForStructWithAlias(model, sm.TableName(), sm.As, sm.IDField())
	for _, f := range includeColumns {
		if _, ok := cols.Cols[f]; !ok {
			return nil, errors.Errorf("Invalid column name %s", f)
		}
		cols.Remove(f)
	}

	xcols := make([]string, len(cols.Cols))
	for n := range cols.Cols {
		xcols = append(xcols, n)
	}
	return xcols, nil
}
