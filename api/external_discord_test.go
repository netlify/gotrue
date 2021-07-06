package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"

	jwt "github.com/dgrijalva/jwt-go"
)

func (ts *ExternalTestSuite) TestSignupExternalDiscord() {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/authorize?provider=discord", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	ts.Require().Equal(http.StatusFound, w.Code)
	u, err := url.Parse(w.Header().Get("Location"))
	ts.Require().NoError(err, "redirect url parse failed")
	q := u.Query()
	ts.Equal(ts.Config.External.Discord.RedirectURI, q.Get("redirect_uri"))
	ts.Equal(ts.Config.External.Discord.ClientID, q.Get("client_id"))
	ts.Equal("code", q.Get("response_type"))
	ts.Equal("email identify ", q.Get("scope"))

	claims := ExternalProviderClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err = p.ParseWithClaims(q.Get("state"), &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(ts.API.config.OperatorToken), nil
	})
	ts.Require().NoError(err)

	ts.Equal("discord", claims.Provider)
	ts.Equal(ts.Config.SiteURL, claims.SiteURL)
}

func DiscordTestSignupSetup(ts *ExternalTestSuite, tokenCount *int, userCount *int, code string, user string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth2/token":
			*tokenCount++
			ts.Equal(code, r.FormValue("code"))
			ts.Equal("authorization_code", r.FormValue("grant_type"))
			ts.Equal(ts.Config.External.Discord.RedirectURI, r.FormValue("redirect_uri"))

			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"discord_token","expires_in":100000}`)
		case "/api/users/@me":
			*userCount++
			w.Header().Add("Content-Type", "application/json")
			fmt.Fprint(w, user)
		default:
			w.WriteHeader(500)
			ts.Fail("unknown discord oauth call %s", r.URL.Path)
		}
	}))

	ts.Config.External.Discord.URL = server.URL

	return server
}

func (ts *ExternalTestSuite) TestSignupExternalDiscord_AuthorizationCode() {
	ts.Config.DisableSignup = false
	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","email":"discord@example.com","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	u := performAuthorization(ts, "discord", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "discord@example.com", "Discord Test", "https://cdn.discordapp.com/avatars/123/abc.png")
}

func (ts *ExternalTestSuite) TestSignupExternalDiscordDisableSignupErrorWhenNoUser() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","email":"discord@example.com","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	u := performAuthorization(ts, "discord", code, "")

	assertAuthorizationFailure(ts, u, "Signups not allowed for this instance", "access_denied", "discord@example.com")
}
func (ts *ExternalTestSuite) TestSignupExternalDiscordDisableSignupErrorWhenEmptyEmail() {
	ts.Config.DisableSignup = true

	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	u := performAuthorization(ts, "discord", code, "")

	assertAuthorizationFailure(ts, u, "Error getting user email from external provider", "server_error", "discord@example.com")
}

func (ts *ExternalTestSuite) TestSignupExternalDiscordDisableSignupSuccessWithPrimaryEmail() {
	ts.Config.DisableSignup = true

	ts.createUser("discord@example.com", "Discord Test", "https://cdn.discordapp.com/avatars/123/abc.png", "")

	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","email":"discord@example.com","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	u := performAuthorization(ts, "discord", code, "")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "discord@example.com", "Discord Test", "https://cdn.discordapp.com/avatars/123/abc.png")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalDiscordSuccessWhenMatchingToken() {
	// name and avatar should be populated from Discord API
	ts.createUser("discord@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","email":"discord@example.com","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	u := performAuthorization(ts, "discord", code, "invite_token")

	assertAuthorizationSuccess(ts, u, tokenCount, userCount, "discord@example.com", "Discord Test", "https://cdn.discordapp.com/avatars/123/abc.png")
}

func (ts *ExternalTestSuite) TestInviteTokenExternalDiscordErrorWhenNoMatchingToken() {
	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","email":"discord@example.com","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "discord", "invite_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalDiscordErrorWhenWrongToken() {
	ts.createUser("discord@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","email":"discord@example.com","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	w := performAuthorizationRequest(ts, "discord", "wrong_token")
	ts.Require().Equal(http.StatusNotFound, w.Code)
}

func (ts *ExternalTestSuite) TestInviteTokenExternalDiscordErrorWhenEmailDoesntMatch() {
	ts.createUser("discord@example.com", "", "", "invite_token")

	tokenCount, userCount := 0, 0
	code := "authcode"
	discordUser := `{"avatar":"abc","email":"other@example.com","id":"123","username":"Discord Test","verified":true}}`
	server := DiscordTestSignupSetup(ts, &tokenCount, &userCount, code, discordUser)
	defer server.Close()

	u := performAuthorization(ts, "discord", code, "invite_token")

	assertAuthorizationFailure(ts, u, "Invited email does not match emails from external provider", "invalid_request", "")
}
