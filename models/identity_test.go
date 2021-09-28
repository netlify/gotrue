package models

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IdentityTestSuite struct {
	suite.Suite
	db *storage.Connection
}

func (ts *IdentityTestSuite) SetupTest() {
	TruncateAll(ts.db)
}

func TestIdentity(t *testing.T) {
	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	conn, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	ts := &IdentityTestSuite{
		db: conn,
	}
	defer ts.db.Close()

	suite.Run(t, ts)
}

func (ts *IdentityTestSuite) TestNewIdentity() {
	u := ts.createUserWithEmail("test@supabase.io")
	ts.Run("Test create identity with no provider id", func() {
		identityData := map[string]interface{}{}
		_, err := NewIdentity(u, "email", identityData)
		require.Error(ts.T(), err, "Error missing provider id")
	})

	ts.Run("Test create identity successfully", func() {
		identityData := map[string]interface{}{"sub": uuid.Nil.String()}
		identity, err := NewIdentity(u, "email", identityData)
		require.NoError(ts.T(), err)
		require.Equal(ts.T(), u.ID, identity.UserID)
	})
}

func (ts *IdentityTestSuite) TestFindUserIdentities() {
	u := ts.createUserWithIdentity("test@supabase.io")
	identities, err := FindIdentitiesByUser(ts.db, u)
	require.NoError(ts.T(), err)

	require.Len(ts.T(), identities, 1)

}

func (ts *IdentityTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser(uuid.Nil, email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	err = ts.db.Create(user)
	require.NoError(ts.T(), err)

	return user
}

func (ts *IdentityTestSuite) createUserWithIdentity(email string) *User {
	user, err := NewUser(uuid.Nil, email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	err = ts.db.Create(user)
	require.NoError(ts.T(), err)

	identityData := map[string]interface{}{
		"sub":   uuid.Nil.String(),
		"name":  "test",
		"email": email,
	}
	require.NoError(ts.T(), err)

	identity, err := NewIdentity(user, "email", identityData)
	require.NoError(ts.T(), err)

	err = ts.db.Create(identity)
	require.NoError(ts.T(), err)

	return user
}
