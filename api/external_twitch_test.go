package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	jwt "github.com/dgrijalva/jwt-go"
)

func (ts *ExternalTestSuite) TestSignupExternalTwitch() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=twitch", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Twitch.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Twitch.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("user:read:email", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.API.config.OperatorToken), nil
	})
	ts.Require().NoError(err)

	ts.Equal("twitch", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func TwitchTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Twitch.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"Twitch_token","expires_in":100000}`)
		case "/helix/users":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown Twitch oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Twitch.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalTwitch_AuthorizationCode() {
	ts.Config.DisableSignup = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch Test","display_name":"Twitch Test","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":"","email":"twitch@example.com"}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()

	u := performAuthorization(ts, "twitch", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "twitch@example.com", "Twitch Test", "https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8")
}

func (ts *ExternalTestSuite) TestSignupExternalTwitchDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch user","display_name":"Twitch user","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":"","email":"twitch@example.com"}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()

	u := performAuthorization(ts, "twitch", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "twitch@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalTwitchDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch user","display_name":"Twitch user","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":""}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()


	u := performAuthorization(ts, "twitch", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "twitch@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalTwitchDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("twitch@example.com", "Twitch Test", "https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch user","display_name":"Twitch user","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":"","email":"twitch@example.com"}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()

	u := performAuthorization(ts, "twitch", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "twitch@example.com", "Twitch Test", "https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalTwitchSuccessWhenMatchingToken() {
	// name and avatar should be populated from Twitch API
	ts.createUser("twitch@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch user","display_name":"Twitch user","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":"","email":"twitch@example.com"}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()

	u := performAuthorization(ts, "twitch", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "twitch@example.com", "Twitch Test", "https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalTwitchErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch user","display_name":"Twitch user","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":"","email":"twitch@example.com"}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "twitch", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalTwitchErrorWhenWrongToken() {
	ts.createUser("twitch@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch user","display_name":"Twitch user","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":"","email":"twitch@example.com"}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "twitch", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalTwitchErrorWhenEmailDoesntMatch() {
	ts.createUser("twitch@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	TwitchUser := `{"data":[{"id":"1","login":"Twitch user","display_name":"Twitch user","type":"","broadcaster_type":"","description":"","profile_image_url":"https://s.gravatar.com/avatar/23463b99b62a72f26ed677cc556c44e8","offline_image_url":"","email":"other@example.com"}]}`
	server := TwitchTestSignupSetup(ts, &tokenCount, &userCount, code, TwitchUser)
	defer server.Close()

	u := performAuthorization(ts, "twitch", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}
