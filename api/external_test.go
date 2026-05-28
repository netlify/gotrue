package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIsAllowedRedirectURI(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		allowed   []string
		want      bool
	}{
		{"exact host", "https://app.example.com/cb", []string{"https://app.example.com"}, true},
		{"empty path on entry matches any candidate path", "https://app.example.com/auth/cb?x=1", []string{"https://app.example.com"}, true},
		{"path prefix", "https://app.example.com/auth/cb", []string{"https://app.example.com/auth"}, true},
		{"path exact", "https://app.example.com/auth", []string{"https://app.example.com/auth"}, true},
		{"path prefix mismatch", "https://app.example.com/other/cb", []string{"https://app.example.com/auth"}, false},
		{"adjacent path prefix not allowed", "https://app.example.com/authorize", []string{"https://app.example.com/auth"}, false},
		{"trailing slash entry", "https://app.example.com/auth/cb", []string{"https://app.example.com/auth/"}, true},
		{"scheme mismatch", "http://app.example.com/cb", []string{"https://app.example.com"}, false},
		{"host mismatch", "https://evil.com/cb", []string{"https://app.example.com"}, false},
		{"subdomain not allowed", "https://evil.app.example.com/cb", []string{"https://app.example.com"}, false},
		{"empty candidate", "", []string{"https://app.example.com"}, false},
		{"candidate without scheme", "/relative/path", []string{"https://app.example.com"}, false},
		{"case-insensitive host", "https://APP.EXAMPLE.COM/cb", []string{"https://app.example.com"}, true},
		{"matches second entry", "https://preview.example.com/cb", []string{"https://app.example.com", "https://preview.example.com"}, true},
		{"empty allowlist", "https://app.example.com/cb", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isAllowedRedirectURI(tc.candidate, tc.allowed))
		})
	}
}

// TestGetExternalRedirectURL_StrictFallbackUsesAllowlist guards against the
// subtle hole where strict mode would still leak a redirect to SiteURL if the
// operator configured AllowedRedirectURIs without including it. With nothing
// matching the allowlist, the result must be an allowlisted entry, not the
// raw SiteURL.
func TestGetExternalRedirectURL_StrictFallbackUsesAllowlist(t *testing.T) {
	api := &API{config: &conf.GlobalConfiguration{}}
	config := &conf.Configuration{
		SiteURL: "https://app.example.com",
		Security: conf.SecurityConfiguration{
			Strict:              true,
			AllowedRedirectURIs: []string{"https://allowed.example.com"},
		},
	}
	config.ApplyDefaults()
	ctx, err := WithInstanceConfig(context.Background(), config, uuid.Nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/callback", nil).WithContext(ctx)
	assert.Equal(t, "https://allowed.example.com", api.getExternalRedirectURL(req))
}

type ExternalTestSuite struct {
	suite.Suite
	API        *API
	Config     *conf.Configuration
	instanceID uuid.UUID
}

func TestExternal(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &ExternalTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *ExternalTestSuite) SetupTest() {
	ts.Config.DisableSignup = false
	ts.Config.Mailer.Autoconfirm = false

	require.NoError(ts.T(), models.TruncateAll(ts.API.db))
}

func (ts *ExternalTestSuite) createUser(email string, name string, avatar string, confirmationToken string) (*models.User, error) {
	// Cleanup existing user, if they already exist
	if u, _ := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, email, ts.Config.JWT.Aud); u != nil {
		require.NoError(ts.T(), ts.API.db.Destroy(u), "Error deleting user")
	}

	u, err := models.NewUser(ts.instanceID, email, "test", ts.Config.JWT.Aud, map[string]interface{}{"full_name": name, "avatar_url": avatar})

	if confirmationToken != "" {
		u.ConfirmationToken = confirmationToken
	}
	ts.Require().NoError(err, "Error making new user")
	ts.Require().NoError(ts.API.db.Create(u), "Error creating user")

	return u, err
}

func performAuthorizationRequest(ts *ExternalTestSuite, provider string, inviteToken string) *httptest.ResponseRecorder {
	authorizeURL := "http://localhost/authorize?provider=" + provider
	if inviteToken != "" {
		authorizeURL = authorizeURL + "&invite_token=" + inviteToken
	}

	req := httptest.NewRequest(http.MethodGet, authorizeURL, nil)
	req.Header.Set("Referer", "https://example.netlify.com/admin")
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	return w
}

func performAuthorization(ts *ExternalTestSuite, provider string, code string, inviteToken string) *url.URL {
	w := performAuthorizationRequest(ts, provider, inviteToken)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	state := q.Get("state")

	// auth server callback
	testURL, err := url.Parse("http://localhost/callback")
	ts.Require().NoError(err)
	v := testURL.Query()
	v.Set("code", code)
	v.Set("state", state)
	testURL.RawQuery = v.Encode()
	req := httptest.NewRequest(http.MethodGet, testURL.String(), nil)
	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err = url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	ts.Require().Equal("/admin", u.Path)

	return u
}

func assertAuthorizationSuccess(ts *ExternalTestSuite, u *url.URL, tokenCount int, userCount int, email string, name string, avatar string) {
	// ensure redirect has #access_token=...
	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().Empty(v.Get("error_description"))
	ts.Require().Empty(v.Get("error"))

	ts.NotEmpty(v.Get("access_token"))
	ts.NotEmpty(v.Get("refresh_token"))
	ts.NotEmpty(v.Get("expires_in"))
	ts.Equal("bearer", v.Get("token_type"))

	ts.Equal(1, tokenCount)
	ts.Equal(1, userCount)

	// ensure user has been created with metadata
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, email, ts.Config.JWT.Aud)
	ts.Require().NoError(err)
	ts.Equal(name, user.UserMetaData["full_name"])
	ts.Equal(avatar, user.UserMetaData["avatar_url"])
}

func assertAuthorizationFailure(ts *ExternalTestSuite, u *url.URL, errorDescription string, errorType string, email string) {
	// ensure new sign ups error
	v, err := url.ParseQuery(u.Fragment)
	ts.Require().NoError(err)
	ts.Require().Equal(errorDescription, v.Get("error_description"))
	ts.Require().Equal(errorType, v.Get("error"))

	ts.Empty(v.Get("access_token"))
	ts.Empty(v.Get("refresh_token"))
	ts.Empty(v.Get("expires_in"))
	ts.Empty(v.Get("token_type"))

	// ensure user is nil
	user, err := models.FindUserByEmailAndAudience(ts.API.db, ts.instanceID, email, ts.Config.JWT.Aud)
	ts.Require().Error(err, "User not found")
	ts.Require().Nil(user)
}

// TestSignupExternalUnsupported tests API /authorize for an unsupported external provider
func (ts *ExternalTestSuite) TestSignupExternalUnsupported() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=external", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Equal(w.Code, http.StatusBadRequest)
}
