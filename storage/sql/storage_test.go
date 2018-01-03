package sql

import (
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var conn *Connection

func TestSQLTestSuite(t *testing.T) {
	config, err := conf.LoadGlobal("../../api/test.env")
	require.NoError(t, err)

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
	conn.db.DropTableIfExists(&models.User{})
	conn.db.DropTableIfExists(&models.RefreshToken{})
	conn.db.DropTableIfExists(&models.Instance{})
	conn.db.CreateTable(&models.User{})
	conn.db.CreateTable(&models.RefreshToken{})
	conn.db.CreateTable(&models.Instance{})
}

func tokenID(r *models.RefreshToken) interface{} {
	return r.ID
}
