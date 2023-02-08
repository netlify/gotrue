package models

import (
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"context"
	"github.com/tigrisdata/tigris-client-go/tigris"
)

type RefreshTokenTestSuite struct {
	suite.Suite
	db *tigris.Database
}

func (ts *RefreshTokenTestSuite) SetupTest() {
	tigris.GetCollection[User](ts.db).DeleteAll(context.TODO())
	tigris.GetCollection[RefreshToken](ts.db).DeleteAll(context.TODO())
	tigris.GetCollection[AuditLogEntry](ts.db).DeleteAll(context.TODO())
}

func TestRefreshToken(t *testing.T) {
	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	tigrisClient, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	database, err := tigrisClient.OpenDatabase(context.TODO(), &User{}, &RefreshToken{}, &AuditLogEntry{})
	require.NoError(t, err)
	ts := &RefreshTokenTestSuite{
		db: database,
	}
	defer tigrisClient.Close()

	suite.Run(t, ts)
}

func (ts *RefreshTokenTestSuite) TestGrantAuthenticatedUser() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(context.TODO(), ts.db, u)
	require.NoError(ts.T(), err)

	require.NotEmpty(ts.T(), r.Token)
	require.Equal(ts.T(), u.ID, r.UserID)
}

func (ts *RefreshTokenTestSuite) TestGrantRefreshTokenSwap() {
	ctx := context.TODO()
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ctx, ts.db, u)
	require.NoError(ts.T(), err)

	s, err := GrantRefreshTokenSwap(ctx, ts.db, u, r)
	require.NoError(ts.T(), err)

	_, nr, err := FindUserWithRefreshToken(ctx, ts.db, r.Token)
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), r.ID, nr.ID)
	require.True(ts.T(), nr.Revoked, "expected old token to be revoked")

	require.NotEqual(ts.T(), r.ID, s.ID)
	require.Equal(ts.T(), u.ID, s.UserID)
}

func (ts *RefreshTokenTestSuite) TestLogout() {
	ctx := context.TODO()
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ctx, ts.db, u)
	require.NoError(ts.T(), err)

	require.NoError(ts.T(), Logout(ctx, ts.db, uuid.Nil, u.ID))
	u, r, err = FindUserWithRefreshToken(ctx, ts.db, r.Token)
	require.Errorf(ts.T(), err, "expected error when there are no refresh tokens to authenticate. user: %v token: %v", u, r)
	require.True(ts.T(), IsNotFoundError(err), "expected NotFoundError")
}

func (ts *RefreshTokenTestSuite) createUser() *User {
	return ts.createUserWithEmail("david@netlify.com")
}

func (ts *RefreshTokenTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser(uuid.Nil, email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	_, err = tigris.GetCollection[User](ts.db).Insert(context.TODO(), user)
	require.NoError(ts.T(), err)

	return user
}
