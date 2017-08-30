package test

import (
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StorageTestSuite struct {
	suite.Suite
	C          storage.Connection
	BeforeTest func()
	TokenID    func(*models.RefreshToken) interface{}
	InstanceID string
}

func (s *StorageTestSuite) SetupTest() {
	s.InstanceID = uuid.NewRandom().String()
	s.BeforeTest()
}

func (s *StorageTestSuite) TestFindUserByConfirmationToken() {
	u := s.createUser()

	n, err := s.C.FindUserByConfirmationToken(u.ConfirmationToken)
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByEmailAndAudience() {
	u := s.createUser()

	n, err := s.C.FindUserByEmailAndAudience(u.InstanceID, u.Email, "test")
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)

	_, err = s.C.FindUserByEmailAndAudience(u.InstanceID, u.Email, "invalid")
	require.EqualError(s.T(), err, models.UserNotFoundError{}.Error())
}

func (s *StorageTestSuite) TestFindUsersInAudience() {
	u := s.createUser()

	n, err := s.C.FindUsersInAudience(u.InstanceID, u.Aud, nil, nil)
	require.NoError(s.T(), err)
	require.Len(s.T(), n, 1)

	p := models.Pagination{
		Page:    1,
		PerPage: 50,
	}
	n, err = s.C.FindUsersInAudience(u.InstanceID, u.Aud, &p, nil)
	require.NoError(s.T(), err)
	require.Len(s.T(), n, 1)
	assert.Equal(s.T(), uint64(1), p.Count)

	sp := &models.SortParams{
		Fields: []models.SortField{
			models.SortField{Name: "created_at", Dir: models.Descending},
		},
	}
	n, err = s.C.FindUsersInAudience(u.InstanceID, u.Aud, nil, sp)
	require.NoError(s.T(), err)
	require.Len(s.T(), n, 1)
}

func (s *StorageTestSuite) TestFindUserByID() {
	u := s.createUser()

	n, err := s.C.FindUserByID(u.ID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByInstanceIDAndID() {
	u := s.createUser()

	n, err := s.C.FindUserByInstanceIDAndID(u.InstanceID, u.ID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByRecoveryToken() {
	u := s.createUser()
	u.GenerateRecoveryToken()

	err := s.C.UpdateUser(u)
	require.NoError(s.T(), err)

	n, err := s.C.FindUserByRecoveryToken(u.RecoveryToken)
	require.NoError(s.T(), err)

	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserWithRefreshToken() {
	u := s.createUser()
	r, err := s.C.GrantAuthenticatedUser(u)
	require.NoError(s.T(), err)

	n, nr, err := s.C.FindUserWithRefreshToken(r.Token)
	require.NoError(s.T(), err)
	require.Equal(s.T(), s.TokenID(r), s.TokenID(nr))
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestGrantAuthenticatedUser() {
	u := s.createUser()
	r, err := s.C.GrantAuthenticatedUser(u)
	require.NoError(s.T(), err)

	require.NotEmpty(s.T(), r.Token)
	require.Equal(s.T(), u.ID, r.UserID)
}

func (s *StorageTestSuite) TestGrantRefreshTokenSwap() {
	u := s.createUser()
	r, err := s.C.GrantAuthenticatedUser(u)
	require.NoError(s.T(), err)

	ts, err := s.C.GrantRefreshTokenSwap(u, r)
	require.NoError(s.T(), err)

	_, nr, err := s.C.FindUserWithRefreshToken(r.Token)
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.TokenID(r), s.TokenID(nr))
	require.True(s.T(), nr.Revoked, "expected old token to be revoked")

	require.NotEqual(s.T(), s.TokenID(r), s.TokenID(ts))
	require.Equal(s.T(), u.ID, ts.UserID)
}

func (s *StorageTestSuite) TestIsDuplicatedEmail() {
	u := s.createUserWithEmail("david.calavera@netlify.com")

	e, err := s.C.IsDuplicatedEmail(u.InstanceID, "david.calavera@netlify.com", "test")
	require.NoError(s.T(), err)
	require.True(s.T(), e, "expected email to be duplicated")

	e, err = s.C.IsDuplicatedEmail(u.InstanceID, "davidcalavera@netlify.com", "test")
	require.NoError(s.T(), err)
	require.False(s.T(), e, "expected email to not be duplicated")

	e, err = s.C.IsDuplicatedEmail(u.InstanceID, "david@netlify.com", "test")
	require.NoError(s.T(), err)
	require.False(s.T(), e, "expected same email to not be duplicated")

	e, err = s.C.IsDuplicatedEmail(u.InstanceID, "david.calavera@netlify.com", "other-aud")
	require.NoError(s.T(), err)
	require.False(s.T(), e, "expected same email to not be duplicated")
}

func (s *StorageTestSuite) TestLogout() {
	u := s.createUser()
	r, err := s.C.GrantAuthenticatedUser(u)
	require.NoError(s.T(), err)

	s.C.Logout(u.ID)
	_, _, err = s.C.FindUserWithRefreshToken(r.Token)
	require.Error(s.T(), err, "expected error when there are no refresh tokens to authenticate")

	require.True(s.T(), models.IsNotFoundError(err), "expected NotFoundError")
}

func (s *StorageTestSuite) TestRevokeToken() {
	u := s.createUser()
	r, err := s.C.GrantAuthenticatedUser(u)
	require.NoError(s.T(), err)

	err = s.C.RevokeToken(r)
	require.NoError(s.T(), err)

	_, nr, err := s.C.FindUserWithRefreshToken(r.Token)
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.TokenID(r), s.TokenID(nr))
	require.True(s.T(), nr.Revoked, "expected token to be revoked")
}

func (s *StorageTestSuite) TestRollbackRefreshTokenSwap() {
	u := s.createUser()
	r, err := s.C.GrantAuthenticatedUser(u)
	require.NoError(s.T(), err)

	ts, err := s.C.GrantRefreshTokenSwap(u, r)
	require.NoError(s.T(), err)

	err = s.C.RollbackRefreshTokenSwap(ts, r)
	require.NoError(s.T(), err)

	_, nr, err := s.C.FindUserWithRefreshToken(r.Token)
	require.NoError(s.T(), err)

	require.False(s.T(), nr.Revoked, "expected token to not be revoked")

	_, ns, err := s.C.FindUserWithRefreshToken(ts.Token)
	require.NoError(s.T(), err)

	require.True(s.T(), ns.Revoked, "expected token to be revoked")
}

func (s *StorageTestSuite) TestUpdateUser() {
	u := s.createUser()

	userUpdates := map[string]interface{}{
		"firstName": "David",
	}
	u.UpdateUserMetaData(userUpdates)

	u.SetRole("admin")

	err := s.C.UpdateUser(u)
	require.NoError(s.T(), err)

	nu, err := s.C.FindUserByInstanceIDAndID(u.InstanceID, u.ID)
	require.NoError(s.T(), err)

	require.NotNil(s.T(), nu.UserMetaData, "expected user metadata to not be nil")

	fn := nu.UserMetaData["firstName"]
	require.Equal(s.T(), "David", fn)

	require.Equal(s.T(), "admin", nu.Role)
}

func (s *StorageTestSuite) TestDeleteUser() {
	u := s.createUserWithEmail("test@example.com")

	require.Equal(s.T(), s.C.DeleteUser(u), nil)

	_, err := s.C.FindUserByEmailAndAudience(u.InstanceID, "test@example.com", "test")
	require.Equal(s.T(), err, models.UserNotFoundError{})
}

func (s *StorageTestSuite) createUser() *models.User {
	return s.createUserWithEmail("david@netlify.com")
}

func (s *StorageTestSuite) createUserWithEmail(email string) *models.User {
	user, err := models.NewUser(s.InstanceID, email, "secret", "test", nil)
	require.NoError(s.T(), err)

	err = s.C.CreateUser(user)
	require.NoError(s.T(), err)

	return user
}
