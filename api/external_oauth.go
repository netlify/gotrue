package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/markbates/goth/gothic"
	"github.com/mrjones/oauth"
	"github.com/netlify/gotrue/api/provider"
	"github.com/sirupsen/logrus"
)

// OAuthProviderData contains the userData and token returned by the oauth provider
type OAuthProviderData struct {
	userData *provider.UserProvidedData
	token    string
}

// loadOAuthState parses the `state` query parameter as a JWS payload,
// extracting the provider requested
func (a *API) loadOAuthState(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	var state string
	if r.Method == http.MethodPost {
		state = r.FormValue("state")
	} else {
		state = r.URL.Query().Get("state")
	}

	if state == "" {
		return nil, badRequestError("OAuth state parameter missing")
	}

	ctx := r.Context()
	oauthToken := r.URL.Query().Get("oauth_token")
	if oauthToken != "" {
		ctx = withRequestToken(ctx, oauthToken)
	}
	oauthVerifier := r.URL.Query().Get("oauth_verifier")
	if oauthVerifier != "" {
		ctx = withOAuthVerifier(ctx, oauthVerifier)
	}
	return a.loadExternalState(ctx, state)
}

func (a *API) oAuthCallback(ctx context.Context, r *http.Request, providerType string) (*OAuthProviderData, error) {
	var rq url.Values
	if err := r.ParseForm(); r.Method == http.MethodPost && err == nil {
		rq = r.Form
	} else {
		rq = r.URL.Query()
	}

	extError := rq.Get("error")
	if extError != "" {
		return nil, oauthError(extError, rq.Get("error_description"))
	}

	oauthCode := rq.Get("code")
	if oauthCode == "" {
		return nil, badRequestError("Authorization code missing")
	}

	oAuthProvider, err := a.OAuthProvider(ctx, providerType)
	if err != nil {
		return nil, badRequestError("Unsupported provider: %+v", err).WithInternalError(err)
	}

	log := getLogEntry(r)
	log.WithFields(logrus.Fields{
		"provider": providerType,
		"code":     oauthCode,
	}).Debug("Exchanging oauth code")

	token, err := oAuthProvider.GetOAuthToken(oauthCode)
	if err != nil {
		return nil, internalServerError("Unable to exchange external code: %s", oauthCode).WithInternalError(err)
	}

	userData, err := oAuthProvider.GetUserData(ctx, token)
	if err != nil {
		return nil, internalServerError("Error getting user email from external provider").WithInternalError(err)
	}

	switch externalProvider := oAuthProvider.(type) {
	case *provider.AppleProvider:
		// apple only returns user info the first time
		oauthUser := rq.Get("user")
		if oauthUser != "" {
			userData.Metadata = externalProvider.ParseUser(oauthUser)
		}
	}

	return &OAuthProviderData{
		userData: userData,
		token:    token.AccessToken,
	}, nil
}

func (a *API) oAuth1Callback(ctx context.Context, r *http.Request, providerType string) (*OAuthProviderData, error) {
	oAuthProvider, err := a.OAuthProvider(ctx, providerType)
	if err != nil {
		return nil, badRequestError("Unsupported provider: %+v", err).WithInternalError(err)
	}
	value, err := gothic.GetFromSession(providerType, r)
	if err != nil {
		return &OAuthProviderData{}, err
	}
	oauthToken := getRequestToken(ctx)
	oauthVerifier := getOAuthVerifier(ctx)
	var accessToken *oauth.AccessToken
	var userData *provider.UserProvidedData
	if twitterProvider, ok := oAuthProvider.(*provider.TwitterProvider); ok {
		requestToken, err := twitterProvider.Unmarshal(value)
		if err != nil {
			return &OAuthProviderData{}, err
		}
		if requestToken.Token != oauthToken {
			return nil, internalServerError("Request token doesn't match token in callback")
		}
		twitterProvider.OauthVerifier = oauthVerifier
		accessToken, err = twitterProvider.Consumer.AuthorizeToken(requestToken, oauthVerifier)
		if err != nil {
			return nil, internalServerError("Unable to retrieve access token").WithInternalError(err)
		}
		userData, err = twitterProvider.FetchUserData(ctx, accessToken)
		if err != nil {
			return nil, internalServerError("Error getting user email from external provider").WithInternalError(err)
		}
	}

	return &OAuthProviderData{
		userData: userData,
		token:    accessToken.Token,
	}, nil

}

// OAuthProvider returns the corresponding oauth provider as an OAuthProvider interface
func (a *API) OAuthProvider(ctx context.Context, name string) (provider.OAuthProvider, error) {
	providerCandidate, err := a.Provider(ctx, name, "")
	if err != nil {
		return nil, err
	}

	switch p := providerCandidate.(type) {
	case provider.OAuthProvider:
		return p, nil
	default:
		return nil, badRequestError("Provider can not be used for OAuth")
	}
}
