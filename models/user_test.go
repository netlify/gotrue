package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tigrisdata/tigris-client-go/tigris"
	"context"
)

const modelsTestConfig = "../hack/test.env"

type UserTestSuite struct {
	suite.Suite
	db *tigris.Database
}

func (ts *UserTestSuite) SetupTest() {
	TruncateAll(ts.db)
}

func TestUser(t *testing.T) {
	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	tigrisClient, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	database, err := tigrisClient.OpenDatabase(context.TODO(), &User{}, &RefreshToken{})
	require.NoError(t, err)

	ts := &UserTestSuite{
		db: database,
	}
	defer tigrisClient.Close()

	suite.Run(t, ts)
}

func (ts *UserTestSuite) TestUpdateAppMetadata() {
	u, err := NewUser(uuid.Nil, "", "", "", nil)
	require.NoError(ts.T(), err)

	ctx := context.TODO()
	require.NoError(ts.T(), u.UpdateAppMetaData(ctx, ts.db, make(map[string]interface{})))

	require.NotNil(ts.T(), u.AppMetaData)

	require.NoError(ts.T(), u.UpdateAppMetaData(ctx, ts.db, map[string]interface{}{
		"foo": "bar",
	}))

	require.Equal(ts.T(), "bar", u.AppMetaData["foo"])
	require.NoError(ts.T(), u.UpdateAppMetaData(ctx, ts.db, map[string]interface{}{
		"foo": nil,
	}))
	require.Len(ts.T(), u.AppMetaData, 0)
	require.Equal(ts.T(), nil, u.AppMetaData["foo"])
}

func (ts *UserTestSuite) TestUpdateUserMetadata() {
	u, err := NewUser(uuid.Nil, "", "", "", nil)
	require.NoError(ts.T(), err)

	ctx := context.TODO()
	require.NoError(ts.T(), u.UpdateUserMetaData(ctx, ts.db, make(map[string]interface{})))

	require.NotNil(ts.T(), u.UserMetaData)

	require.NoError(ts.T(), u.UpdateUserMetaData(ctx, ts.db, map[string]interface{}{
		"foo": "bar",
	}))

	require.Equal(ts.T(), "bar", u.UserMetaData["foo"])
	require.NoError(ts.T(), u.UpdateUserMetaData(ctx, ts.db, map[string]interface{}{
		"foo": nil,
	}))
	require.Len(ts.T(), u.UserMetaData, 0)
	require.Equal(ts.T(), nil, u.UserMetaData["foo"])
}

func (ts *UserTestSuite) TestFindUserByConfirmationToken() {
	u := ts.createUser()

	n, err := FindUserByConfirmationToken(context.TODO(), ts.db, u.ConfirmationToken)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByEmailAndAudience() {
	u := ts.createUser()

	n, err := FindUserByEmailAndAudience(context.TODO(), ts.db, u.InstanceID, u.Email, "test")
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)

	_, err = FindUserByEmailAndAudience(context.TODO(), ts.db, u.InstanceID, u.Email, "invalid")
	require.EqualError(ts.T(), err, "document not found")
}

func (ts *UserTestSuite) TestFindUsersInAudience() {
	u := ts.createUser()

	ctx := context.TODO()
	n, err := FindUsersInAudience(ctx, ts.db, u.InstanceID, u.Aud, nil, nil, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)

	p := Pagination{
		Page:    1,
		PerPage: 50,
	}
	n, err = FindUsersInAudience(ctx, ts.db, u.InstanceID, u.Aud, &p, nil, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)
	//ToDo: pagination related
	//assert.Equal(ts.T(), uint64(1), p.Count)

	sp := &SortParams{
		Fields: []SortField{
			SortField{Name: "created_at", Dir: Descending},
		},
	}
	n, err = FindUsersInAudience(ctx, ts.db, u.InstanceID, u.Aud, nil, sp, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)
}

func (ts *UserTestSuite) TestFindUserByID() {
	u := ts.createUser()

	n, err := FindUserByID(context.TODO(), ts.db, u.ID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByInstanceIDAndID() {
	u := ts.createUser()

	n, err := FindUserByInstanceIDAndID(context.TODO(), ts.db, u.InstanceID, u.ID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByRecoveryToken() {
	u := ts.createUser()
	u.RecoveryToken = "asdf"

	ctx := context.TODO()
	_, err := tigris.GetCollection[User](ts.db).InsertOrReplace(ctx, u)
	require.NoError(ts.T(), err)

	n, err := FindUserByRecoveryToken(ctx, ts.db, u.RecoveryToken)
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserWithRefreshToken() {
	u := ts.createUser()

	ctx := context.TODO()
	r, err := GrantAuthenticatedUser(ctx, ts.db, u)
	require.NoError(ts.T(), err)

	n, nr, err := FindUserWithRefreshToken(ctx, ts.db, r.Token)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), r.ID, nr.ID)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestIsDuplicatedEmail() {
	u := ts.createUserWithEmail("himank.chaudhary@tigrisdata.com")

	ctx := context.TODO()
	e, err := IsDuplicatedEmail(ctx, ts.db, u.InstanceID, "himank.chaudhary@tigrisdata.com", "test")
	require.NoError(ts.T(), err)
	require.True(ts.T(), e, "expected email to be duplicated")

	e, err = IsDuplicatedEmail(ctx, ts.db, u.InstanceID, "himankchaudhary@tigrisdata.com", "test")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected email to not be duplicated")

	e, err = IsDuplicatedEmail(ctx, ts.db, u.InstanceID, "himank@tigrisdata.com", "test")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected same email to not be duplicated")

	e, err = IsDuplicatedEmail(ctx, ts.db, u.InstanceID, "himank.chaudhary@tigrisdata.com", "other-aud")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected same email to not be duplicated")
}

func (ts *UserTestSuite) createUser() *User {
	return ts.createUserWithEmail("himank@tigrisdata.com")
}

func (ts *UserTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser(uuid.Nil, email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	_, err = tigris.GetCollection[User](ts.db).Insert(context.TODO(), user)
	require.NoError(ts.T(), err)

	return user
}
