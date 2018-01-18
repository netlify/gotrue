package sql

import (
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StorageTestSuite struct {
	suite.Suite
	C          *storage.Connection
	TokenID    func(*models.RefreshToken) interface{}
	InstanceID uuid.UUID
}

func TestSQLTestSuite(t *testing.T) {
	config, err := conf.LoadGlobal("../../hack/test.env")
	require.NoError(t, err)

	conn, err := storage.Dial(config)
	require.NoError(t, err)

	s := &StorageTestSuite{
		C:       conn,
		TokenID: tokenID,
	}
	defer conn.Close()
	suite.Run(t, s)
}

func tokenID(r *models.RefreshToken) interface{} {
	return r.ID
}

func (s *StorageTestSuite) SetupTest() {
	s.InstanceID = uuid.Must(uuid.NewV4())

	models.TruncateAll(s.C)
}

func (s *StorageTestSuite) TestFindUserByConfirmationToken() {
	u := s.createUser()

	n, err := models.FindUserByConfirmationToken(s.C, u.ConfirmationToken)
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByEmailAndAudience() {
	u := s.createUser()

	n, err := models.FindUserByEmailAndAudience(s.C, u.InstanceID, u.Email, "test")
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)

	_, err = models.FindUserByEmailAndAudience(s.C, u.InstanceID, u.Email, "invalid")
	require.EqualError(s.T(), err, models.UserNotFoundError{}.Error())
}

func (s *StorageTestSuite) TestFindUsersInAudience() {
	u := s.createUser()

	n, err := models.FindUsersInAudience(s.C, u.InstanceID, u.Aud, nil, nil, "")
	require.NoError(s.T(), err)
	require.Len(s.T(), n, 1)

	p := models.Pagination{
		Page:    1,
		PerPage: 50,
	}
	n, err = models.FindUsersInAudience(s.C, u.InstanceID, u.Aud, &p, nil, "")
	require.NoError(s.T(), err)
	require.Len(s.T(), n, 1)
	assert.Equal(s.T(), uint64(1), p.Count)

	sp := &models.SortParams{
		Fields: []models.SortField{
			models.SortField{Name: "created_at", Dir: models.Descending},
		},
	}
	n, err = models.FindUsersInAudience(s.C, u.InstanceID, u.Aud, nil, sp, "")
	require.NoError(s.T(), err)
	require.Len(s.T(), n, 1)
}

func (s *StorageTestSuite) TestFindUserByID() {
	u := s.createUser()

	n, err := models.FindUserByID(s.C, u.ID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByInstanceIDAndID() {
	u := s.createUser()

	n, err := models.FindUserByInstanceIDAndID(s.C, u.InstanceID, u.ID)
	require.NoError(s.T(), err)
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserByRecoveryToken() {
	u := s.createUser()
	u.RecoveryToken = "asdf"

	err := s.C.Update(u)
	require.NoError(s.T(), err)

	n, err := models.FindUserByRecoveryToken(s.C, u.RecoveryToken)
	require.NoError(s.T(), err)

	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestFindUserWithRefreshToken() {
	u := s.createUser()
	r, err := models.GrantAuthenticatedUser(s.C, u)
	require.NoError(s.T(), err)

	n, nr, err := models.FindUserWithRefreshToken(s.C, r.Token)
	require.NoError(s.T(), err)
	require.Equal(s.T(), s.TokenID(r), s.TokenID(nr))
	require.Equal(s.T(), u.ID, n.ID)
}

func (s *StorageTestSuite) TestGrantAuthenticatedUser() {
	u := s.createUser()
	r, err := models.GrantAuthenticatedUser(s.C, u)
	require.NoError(s.T(), err)

	require.NotEmpty(s.T(), r.Token)
	require.Equal(s.T(), u.ID, r.UserID)
}

func (s *StorageTestSuite) TestGrantRefreshTokenSwap() {
	u := s.createUser()
	r, err := models.GrantAuthenticatedUser(s.C, u)
	require.NoError(s.T(), err)

	ts, err := models.GrantRefreshTokenSwap(s.C, u, r)
	require.NoError(s.T(), err)

	_, nr, err := models.FindUserWithRefreshToken(s.C, r.Token)
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.TokenID(r), s.TokenID(nr))
	require.True(s.T(), nr.Revoked, "expected old token to be revoked")

	require.NotEqual(s.T(), s.TokenID(r), s.TokenID(ts))
	require.Equal(s.T(), u.ID, ts.UserID)
}

func (s *StorageTestSuite) TestIsDuplicatedEmail() {
	u := s.createUserWithEmail("david.calavera@netlify.com")

	e, err := models.IsDuplicatedEmail(s.C, u.InstanceID, "david.calavera@netlify.com", "test")
	require.NoError(s.T(), err)
	require.True(s.T(), e, "expected email to be duplicated")

	e, err = models.IsDuplicatedEmail(s.C, u.InstanceID, "davidcalavera@netlify.com", "test")
	require.NoError(s.T(), err)
	require.False(s.T(), e, "expected email to not be duplicated")

	e, err = models.IsDuplicatedEmail(s.C, u.InstanceID, "david@netlify.com", "test")
	require.NoError(s.T(), err)
	require.False(s.T(), e, "expected same email to not be duplicated")

	e, err = models.IsDuplicatedEmail(s.C, u.InstanceID, "david.calavera@netlify.com", "other-aud")
	require.NoError(s.T(), err)
	require.False(s.T(), e, "expected same email to not be duplicated")
}

func (s *StorageTestSuite) TestLogout() {
	u := s.createUser()
	r, err := models.GrantAuthenticatedUser(s.C, u)
	require.NoError(s.T(), err)

	models.Logout(s.C, u.ID)
	u, r, err = models.FindUserWithRefreshToken(s.C, r.Token)
	require.Error(s.T(), err, "expected error when there are no refresh tokens to authenticate. user: %v token: %v", u, r)

	require.True(s.T(), models.IsNotFoundError(err), "expected NotFoundError")
}

func (s *StorageTestSuite) TestRevokeToken() {
	u := s.createUser()
	r, err := models.GrantAuthenticatedUser(s.C, u)
	require.NoError(s.T(), err)

	err = models.RevokeToken(s.C, r)
	require.NoError(s.T(), err)

	_, nr, err := models.FindUserWithRefreshToken(s.C, r.Token)
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.TokenID(r), s.TokenID(nr))
	require.True(s.T(), nr.Revoked, "expected token to be revoked")
}

func (s *StorageTestSuite) TestRollbackRefreshTokenSwap() {
	u := s.createUser()
	r, err := models.GrantAuthenticatedUser(s.C, u)
	require.NoError(s.T(), err)

	ts, err := models.GrantRefreshTokenSwap(s.C, u, r)
	require.NoError(s.T(), err)

	err = models.RollbackRefreshTokenSwap(s.C, ts, r)
	require.NoError(s.T(), err)

	_, nr, err := models.FindUserWithRefreshToken(s.C, r.Token)
	require.NoError(s.T(), err)

	require.False(s.T(), nr.Revoked, "expected token to not be revoked")

	_, ns, err := models.FindUserWithRefreshToken(s.C, ts.Token)
	require.NoError(s.T(), err)

	require.True(s.T(), ns.Revoked, "expected token to be revoked")
}

func (s *StorageTestSuite) TestUpdateUser() {
	u := s.createUser()

	userUpdates := map[string]interface{}{
		"firstName": "David",
	}
	u.UpdateUserMetaData(s.C, userUpdates)

	u.SetRole(s.C, "admin")

	err := s.C.Update(u)
	require.NoError(s.T(), err)

	nu, err := models.FindUserByInstanceIDAndID(s.C, u.InstanceID, u.ID)
	require.NoError(s.T(), err)

	require.NotNil(s.T(), nu.UserMetaData, "expected user metadata to not be nil")

	fn := nu.UserMetaData["firstName"]
	require.Equal(s.T(), "David", fn)

	require.Equal(s.T(), "admin", nu.Role)
}

func (s *StorageTestSuite) TestDeleteUser() {
	u := s.createUserWithEmail("test@example.com")

	require.Equal(s.T(), s.C.Destroy(u), nil)

	_, err := models.FindUserByEmailAndAudience(s.C, u.InstanceID, "test@example.com", "test")
	require.Equal(s.T(), err, models.UserNotFoundError{})
}

func (s *StorageTestSuite) createUser() *models.User {
	return s.createUserWithEmail("david@netlify.com")
}

func (s *StorageTestSuite) createUserWithEmail(email string) *models.User {
	user, err := models.NewUser(s.InstanceID, email, "secret", "test", nil)
	require.NoError(s.T(), err)

	err = s.C.Create(user)
	require.NoError(s.T(), err)

	return user
}
