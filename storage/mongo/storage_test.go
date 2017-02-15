package mongo

import (
	"fmt"
	"os"
	"testing"

	"github.com/netlify/netlify-auth/conf"
	"github.com/netlify/netlify-auth/models"
)

var conn *Connection

func TestMain(m *testing.M) {
	connURL := os.Getenv("NETLIFY_AUTH_MONGODB_TEST_CONN_URL")

	if connURL == "" {
		fmt.Println(`MongoDB test suite disabled.
Set the environment variable NETLIFY_AUTH_MONGODB_TEST_CONN_URL with the connection URL to enable them.`)
		return
	}

	config := &conf.Configuration{
		DB: conf.DBConfiguration{
			Namespace: "netlify_auth",
			Driver:    "mongodb",
			ConnURL:   connURL,
		},
		JWT: conf.JWTConfiguration{
			AdminGroupName: "admin-test",
		},
	}

	var err error
	conn, err = Connect(config)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestCreateFirstUser(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	if !u.HasRole("admin-test") {
		t.Fatalf("expected first user to be an admin, got %v", u.AppMetaData)
	}

	n := createUserWithEmail(t, "david.calavera@netlify.com")
	if n.HasRole("admin-test") {
		t.Fatal("expected second user to not be an admin")
	}
}

func TestFindUserByConfirmationToken(t *testing.T) {
	defer cleanTables()
	u := createUser(t)

	n, err := conn.FindUserByConfirmationToken(u.ConfirmationToken)
	if err != nil {
		t.Fatal(err)
	}

	if n.ID != u.ID {
		t.Fatalf("expected %q, got %q", u, n)
	}
}

func TestFindUserByEmail(t *testing.T) {
	defer cleanTables()
	u := createUser(t)

	n, err := conn.FindUserByEmail("david@netlify.com")
	if err != nil {
		t.Fatal(err)
	}

	if n.ID != u.ID {
		t.Fatalf("expected %q, got %q", u, n)
	}
}

func TestFindUserByID(t *testing.T) {
	defer cleanTables()
	u := createUser(t)

	n, err := conn.FindUserByID(u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if n.ID != u.ID {
		t.Fatalf("expected %q, got %q", u, n)
	}
}

func TestFindUserByRecoveryToken(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	u.GenerateRecoveryToken()
	if err := conn.UpdateUser(u); err != nil {
		t.Fatal(err)
	}

	n, err := conn.FindUserByRecoveryToken(u.RecoveryToken)
	if err != nil {
		t.Fatal(err)
	}

	if n.ID != u.ID {
		t.Fatalf("expected %q, got %q", u, n)
	}
}

func TestFindUserWithRefreshToken(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	r, err := conn.GrantAuthenticatedUser(u)
	if err != nil {
		t.Fatal(err)
	}

	n, nr, err := conn.FindUserWithRefreshToken(r.Token)
	if err != nil {
		t.Fatal(err)
	}

	if nr.BID != r.BID {
		t.Fatalf("expected %q, got %q", r, nr)
	}

	if n.ID != u.ID {
		t.Fatalf("expected %q, got %q", u, n)
	}
}

func TestGrantAuthenticatedUser(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	r, err := conn.GrantAuthenticatedUser(u)
	if err != nil {
		t.Fatal(err)
	}

	if r.Token == "" {
		t.Fatal("expected token to not be an empty string")
	}

	if r.UserID != u.ID {
		t.Fatalf("expected token assigned to %v, got %v", u.ID, r.UserID)
	}
}

func TestGrantRefreshTokenSwap(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	r, err := conn.GrantAuthenticatedUser(u)
	if err != nil {
		t.Fatal(err)
	}

	s, err := conn.GrantRefreshTokenSwap(u, r)
	if err != nil {
		t.Fatal(err)
	}

	_, nr, err := conn.FindUserWithRefreshToken(r.Token)
	if err != nil {
		t.Fatal(err)
	}

	if nr.BID != r.BID {
		t.Fatalf("expected %q, got %q", r, nr)
	}

	if !nr.Revoked {
		t.Fatal("expected old token to be revoked")
	}

	if s.BID == r.BID {
		t.Fatalf("expected a new token %q, got %q", s, r)
	}

	if s.UserID != u.ID {
		t.Fatalf("expected token assigned to %v, got %v", u.ID, s.UserID)
	}
}

func TestIsDuplicatedEmail(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	createUserWithEmail(t, "david.calavera@netlify.com")

	e, err := conn.IsDuplicatedEmail("david.calavera@netlify.com", u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !e {
		t.Fatal("expected email to be duplicated")
	}

	e, err = conn.IsDuplicatedEmail("davidcalavera@netlify.com", u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if e {
		t.Fatal("expected email to not be duplicated")
	}

	e, err = conn.IsDuplicatedEmail("david@netlify.com", u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if e {
		t.Fatal("expected same email to not be duplicated")
	}
}

func TestLogout(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	r, err := conn.GrantAuthenticatedUser(u)
	if err != nil {
		t.Fatal(err)
	}

	conn.Logout(u.ID)
	_, _, err = conn.FindUserWithRefreshToken(r.Token)
	if err == nil {
		t.Fatal("expected error when there are no refresh tokens to authenticate")
	}

	if !models.IsNotFoundError(err) {
		t.Fatal("expected NotFoundError, got %q", err)
	}
}

func TestRevokeToken(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	r, err := conn.GrantAuthenticatedUser(u)
	if err != nil {
		t.Fatal(err)
	}

	err = conn.RevokeToken(r)
	if err != nil {
		t.Fatal(err)
	}

	_, nr, err := conn.FindUserWithRefreshToken(r.Token)
	if err != nil {
		t.Fatal(err)
	}

	if nr.BID != r.BID {
		t.Fatalf("expected %q, got %q", r, nr)
	}

	if !nr.Revoked {
		t.Fatal("expected token to be revoked")
	}
}

func TestRollbackRefreshTokenSwap(t *testing.T) {
	defer cleanTables()
	u := createUser(t)
	r, err := conn.GrantAuthenticatedUser(u)
	if err != nil {
		t.Fatal(err)
	}

	s, err := conn.GrantRefreshTokenSwap(u, r)
	if err != nil {
		t.Fatal(err)
	}

	err = conn.RollbackRefreshTokenSwap(s, r)
	if err != nil {
		t.Fatal(err)
	}

	_, nr, err := conn.FindUserWithRefreshToken(r.Token)
	if err != nil {
		t.Fatal(err)
	}

	if nr.Revoked {
		t.Fatal("expected token to be not revoked")
	}

	_, ns, err := conn.FindUserWithRefreshToken(s.Token)
	if err != nil {
		t.Fatal(err)
	}

	if !ns.Revoked {
		t.Fatal("expected token to be revoked")
	}
}

func TestUpdateUser(t *testing.T) {
	defer cleanTables()
	u := createUser(t)

	userUpdates := map[string]interface{}{
		"firstName": "David",
	}
	u.UpdateUserMetaData(userUpdates)

	appUpdates := map[string]interface{}{
		"roles": []string{"admin"},
	}
	u.UpdateAppMetaData(appUpdates)

	if err := conn.UpdateUser(u); err != nil {
		t.Fatal(err)
	}

	nu, err := conn.FindUserByID(u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if nu.UserMetaData == nil {
		t.Fatal("expected user metadata to not be nil")
	}

	if fn := nu.UserMetaData["firstName"]; fn != "David" {
		t.Fatalf("expected %v, got %v", "David", fn)
	}

	if nu.AppMetaData == nil {
		t.Fatal("expected app metadata to not be nil")
	}

	rr := nu.AppMetaData["roles"]
	if rr == nil {
		t.Fatal("expected roles to not be nil")
	}

	roles := rr.([]interface{})
	if roles[0].(string) != "admin" {
		t.Fatalf("expected %v, got %v", "admin", roles[0])
	}
}

func createUser(t *testing.T) *models.User {
	return createUserWithEmail(t, "david@netlify.com")
}

func createUserWithEmail(t *testing.T, email string) *models.User {
	user, err := models.NewUser(email, "secret", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := conn.CreateUser(user); err != nil {
		t.Fatal(err)
	}

	return user
}

func cleanTables() {
	u := &models.User{}
	r := &models.RefreshToken{}
	conn.db.C(u.TableName()).DropCollection()
	conn.db.C(r.TableName()).DropCollection()
}
