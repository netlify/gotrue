package mongo

import (
	"os"
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var conn *Connection

func TestMongoDBTestSuite(t *testing.T) {
	connURL := os.Getenv("GOTRUE_MONGODB_TEST_CONN_URL")

	if connURL == "" {
		t.Skip(`MongoDB test suite disabled.
Set the environment variable GOTRUE_MONGODB_TEST_CONN_URL with the connection URL to enable them.`)
	}

	config := &conf.Configuration{
		DB: conf.DBConfiguration{
			Namespace: "gotrue",
			Driver:    "mongodb",
			ConnURL:   connURL,
		},
		JWT: conf.JWTConfiguration{
			AdminGroupName: "admin-test",
		},
	}

	var err error
	conn, err = Dial(config)
	require.NoError(t, err)

	s := &test.StorageTestSuite{
		C:          conn,
		BeforeTest: beforeTest,
		TokenID:    tokenID,
	}
	suite.Run(t, s)
}

func beforeTest() {
	u := &models.User{}
	r := &models.RefreshToken{}
	conn.db.C(u.TableName()).DropCollection()
	conn.db.C(r.TableName()).DropCollection()
}

func tokenID(r *models.RefreshToken) interface{} {
	return r.BID
}
