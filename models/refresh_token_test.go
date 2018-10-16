package models

import (
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/gobuffalo/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RefreshTokenTestSuite struct {
	suite.Suite
	db *storage.Connection
}

func (ts *RefreshTokenTestSuite) SetupTest() {
	TruncateAll(ts.db)
}

func TestRefreshToken(t *testing.T) {
	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	conn, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	ts := &RefreshTokenTestSuite{
		db: conn,
	}
	defer ts.db.Close()

	suite.Run(t, ts)
}

func (ts *RefreshTokenTestSuite) TestGrantAuthenticatedUser() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u)
	require.NoError(ts.T(), err)

	require.NotEmpty(ts.T(), r.Token)
	require.Equal(ts.T(), u.ID, r.UserID)
}

func (ts *RefreshTokenTestSuite) TestGrantRefreshTokenSwap() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u)
	require.NoError(ts.T(), err)

	s, err := GrantRefreshTokenSwap(ts.db, u, r)
	require.NoError(ts.T(), err)

	_, nr, err := FindUserWithRefreshToken(ts.db, r.Token)
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), r.ID, nr.ID)
	require.True(ts.T(), nr.Revoked, "expected old token to be revoked")

	require.NotEqual(ts.T(), r.ID, s.ID)
	require.Equal(ts.T(), u.ID, s.UserID)
}

func (ts *RefreshTokenTestSuite) TestLogout() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u)
	require.NoError(ts.T(), err)

	require.NoError(ts.T(), Logout(ts.db, uuid.Nil, u.ID))
	u, r, err = FindUserWithRefreshToken(ts.db, r.Token)
	require.Errorf(ts.T(), err, "expected error when there are no refresh tokens to authenticate. user: %v token: %v", u, r)

	require.True(ts.T(), IsNotFoundError(err), "expected NotFoundError")
}

func (ts *RefreshTokenTestSuite) createUser() *User {
	return ts.createUserWithEmail("david@netlify.com")
}

func (ts *RefreshTokenTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser(uuid.Nil, email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	err = ts.db.Create(user)
	require.NoError(ts.T(), err)

	return user
}
