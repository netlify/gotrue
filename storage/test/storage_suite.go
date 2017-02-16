package test

import (
	"github.com/netlify/netlify-auth/models"
	"github.com/netlify/netlify-auth/storage"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StorageTestSuite struct {
	suite.Suite
	C          storage.Connection
	BeforeTest func()
	TokenID    func(*models.RefreshToken) interface{}
}

func (s *StorageTestSuite) SetupTest() {
	s.BeforeTest()
}

func (s *StorageTestSuite) TestCreateFirstUser() {
	u := s.createUser()
	require.True(s.T(), u.HasRole("admin-test"), "expected first user to be an admin", u.AppMetaData)

	n := s.createUserWithEmail("david.calavera@netlify.com")
	require.False(s.T(), n.HasRole("admin-test"), "expected second user to not be an admin")
}

func (s *StorageTestSuite) TestFindUserByConfirmationToken() {
	u := s.createUser()

	n, err := s.C.FindUserByConfirmationToken(u.ConfirmationToken)
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByEmail() {
	u := s.createUser()

	n, err := s.C.FindUserByEmail("david@netlify.com")
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByID() {
	u := s.createUser()

	n, err := s.C.FindUserByID(u.ID)
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
	u := s.createUser()
	s.createUserWithEmail("david.calavera@netlify.com")

	e, err := s.C.IsDuplicatedEmail("david.calavera@netlify.com", u.ID)
	require.NoError(s.T(), err)

	require.True(s.T(), e, "expected email to be duplicated")

	e, err = s.C.IsDuplicatedEmail("davidcalavera@netlify.com", u.ID)
	require.NoError(s.T(), err)

	require.False(s.T(), e, "expected email to not be duplicated")

	e, err = s.C.IsDuplicatedEmail("david@netlify.com", u.ID)
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

	appUpdates := map[string]interface{}{
		"roles": []string{"admin"},
	}
	u.UpdateAppMetaData(appUpdates)

	err := s.C.UpdateUser(u)
	require.NoError(s.T(), err)

	nu, err := s.C.FindUserByID(u.ID)
	require.NoError(s.T(), err)

	require.NotNil(s.T(), nu.UserMetaData, "expected user metadata to not be nil")

	fn := nu.UserMetaData["firstName"]
	require.Equal(s.T(), "David", fn)

	require.NotNil(s.T(), nu.AppMetaData, "expected app metadata to not be nil")

	rr := nu.AppMetaData["roles"]
	require.NotNil(s.T(), rr, "expected roles to not be nil")

	roles := rr.([]interface{})
	require.Equal(s.T(), "admin", roles[0].(string))
}

func (s *StorageTestSuite) createUser() *models.User {
	return s.createUserWithEmail("david@netlify.com")
}

func (s *StorageTestSuite) createUserWithEmail(email string) *models.User {
	user, err := models.NewUser(email, "secret", nil)
	require.NoError(s.T(), err)

	err = s.C.CreateUser(user)
	require.NoError(s.T(), err)

	return user
}
