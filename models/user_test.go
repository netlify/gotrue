package models

import (
	"testing"

	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/storage"
	"github.com/netlify/gotrue/storage/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const modelsTestConfig = "../hack/test.env"

type UserTestSuite struct {
	suite.Suite
	db *storage.Connection
}

func (ts *UserTestSuite) SetupTest() {
	TruncateAll(ts.db)
}

func TestUser(t *testing.T) {
	globalConfig, err := conf.LoadGlobal(modelsTestConfig)
	require.NoError(t, err)

	conn, err := test.SetupDBConnection(globalConfig)
	require.NoError(t, err)

	ts := &UserTestSuite{
		db: conn,
	}
	defer ts.db.Close()

	suite.Run(t, ts)
}

func (ts *UserTestSuite) TestUpdateAppMetadata() {
	u, err := NewUser(uuid.Nil, "", "", "", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), u.UpdateAppMetaData(ts.db, make(map[string]interface{})))

	require.NotNil(ts.T(), u.AppMetaData)

	require.NoError(ts.T(), u.UpdateAppMetaData(ts.db, map[string]interface{}{
		"foo": "bar",
	}))

	require.Equal(ts.T(), "bar", u.AppMetaData["foo"])
	require.NoError(ts.T(), u.UpdateAppMetaData(ts.db, map[string]interface{}{
		"foo": nil,
	}))
	require.Len(ts.T(), u.AppMetaData, 0)
	require.Equal(ts.T(), nil, u.AppMetaData["foo"])
}

func (ts *UserTestSuite) TestUpdateUserMetadata() {
	u, err := NewUser(uuid.Nil, "", "", "", nil)
	require.NoError(ts.T(), err)
	require.NoError(ts.T(), u.UpdateUserMetaData(ts.db, make(map[string]interface{})))

	require.NotNil(ts.T(), u.UserMetaData)

	require.NoError(ts.T(), u.UpdateUserMetaData(ts.db, map[string]interface{}{
		"foo": "bar",
	}))

	require.Equal(ts.T(), "bar", u.UserMetaData["foo"])
	require.NoError(ts.T(), u.UpdateUserMetaData(ts.db, map[string]interface{}{
		"foo": nil,
	}))
	require.Len(ts.T(), u.UserMetaData, 0)
	require.Equal(ts.T(), nil, u.UserMetaData["foo"])
}

func (ts *UserTestSuite) TestFindUserByConfirmationToken() {
	u := ts.createUser()

	n, err := FindUserByConfirmationToken(ts.db, u.ConfirmationToken)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByEmailAndAudience() {
	u := ts.createUser()

	n, err := FindUserByEmailAndAudience(ts.db, u.InstanceID, u.Email, "test")
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)

	_, err = FindUserByEmailAndAudience(ts.db, u.InstanceID, u.Email, "invalid")
	require.EqualError(ts.T(), err, UserNotFoundError{}.Error())
}

func (ts *UserTestSuite) TestFindUsersInAudience() {
	u := ts.createUser()

	n, err := FindUsersInAudience(ts.db, u.InstanceID, u.Aud, nil, nil, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)

	p := Pagination{
		Page:    1,
		PerPage: 50,
	}
	n, err = FindUsersInAudience(ts.db, u.InstanceID, u.Aud, &p, nil, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)
	assert.Equal(ts.T(), uint64(1), p.Count)

	sp := &SortParams{
		Fields: []SortField{
			SortField{Name: "created_at", Dir: Descending},
		},
	}
	n, err = FindUsersInAudience(ts.db, u.InstanceID, u.Aud, nil, sp, "")
	require.NoError(ts.T(), err)
	require.Len(ts.T(), n, 1)
}

func (ts *UserTestSuite) TestFindUserByID() {
	u := ts.createUser()

	n, err := FindUserByID(ts.db, u.ID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByInstanceIDAndID() {
	u := ts.createUser()

	n, err := FindUserByInstanceIDAndID(ts.db, u.InstanceID, u.ID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserByRecoveryToken() {
	u := ts.createUser()
	u.RecoveryToken = "asdf"

	err := ts.db.Update(u)
	require.NoError(ts.T(), err)

	n, err := FindUserByRecoveryToken(ts.db, u.RecoveryToken)
	require.NoError(ts.T(), err)

	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestFindUserWithRefreshToken() {
	u := ts.createUser()
	r, err := GrantAuthenticatedUser(ts.db, u)
	require.NoError(ts.T(), err)

	n, nr, err := FindUserWithRefreshToken(ts.db, r.Token)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), r.ID, nr.ID)
	require.Equal(ts.T(), u.ID, n.ID)
}

func (ts *UserTestSuite) TestIsDuplicatedEmail() {
	u := ts.createUserWithEmail("david.calavera@netlify.com")

	e, err := IsDuplicatedEmail(ts.db, u.InstanceID, "david.calavera@netlify.com", "test")
	require.NoError(ts.T(), err)
	require.True(ts.T(), e, "expected email to be duplicated")

	e, err = IsDuplicatedEmail(ts.db, u.InstanceID, "davidcalavera@netlify.com", "test")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected email to not be duplicated")

	e, err = IsDuplicatedEmail(ts.db, u.InstanceID, "david@netlify.com", "test")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected same email to not be duplicated")

	e, err = IsDuplicatedEmail(ts.db, u.InstanceID, "david.calavera@netlify.com", "other-aud")
	require.NoError(ts.T(), err)
	require.False(ts.T(), e, "expected same email to not be duplicated")
}

func (ts *UserTestSuite) createUser() *User {
	return ts.createUserWithEmail("david@netlify.com")
}

func (ts *UserTestSuite) createUserWithEmail(email string) *User {
	user, err := NewUser(uuid.Nil, email, "secret", "test", nil)
	require.NoError(ts.T(), err)

	err = ts.db.Create(user)
	require.NoError(ts.T(), err)

	return user
}
