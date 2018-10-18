package api

import (
	"context"
	"net/http"

	"github.com/netlify/gotrue/api/provider"
	"github.com/netlify/gotrue/api/provider/oauth2"
	"github.com/sirupsen/logrus"
)

// loadOAuthState parses the `state` query parameter as a JWS payload,
// extracting the provider requested
func (a *API) loadOAuthState(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return nil, badRequestError("OAuth state parameter missing")
	}

	ctx := r.Context()
	return a.loadExternalState(ctx, state)
}

func (a *API) oAuthCallback(ctx context.Context, r *http.Request, providerType string) (*provider.UserProvidedData, error) {
	rq := r.URL.Query()

	extError := rq.Get("error")
	if extError != "" {
		return nil, oauthError(extError, rq.Get("error_description"))
	}

	oauthCode := rq.Get("code")
	if oauthCode == "" {
		return nil, badRequestError("Authorization code missing")
	}

	oAuth2Provider, err := a.OAuth2Provider(ctx, providerType)
	if err != nil {
		return nil, badRequestError("Unsupported provider: %+v", err).WithInternalError(err)
	}

	log := getLogEntry(r)
	log.WithFields(logrus.Fields{
		"provider": providerType,
		"code":     oauthCode,
	}).Debug("Exchanging oauth code")

	tok, err := oAuth2Provider.GetOAuthToken(oauthCode)
	if err != nil {
		return nil, internalServerError("Unable to exchange external code: %s", oauthCode).WithInternalError(err)
	}

	userData, err := oAuth2Provider.GetUserData(ctx, tok)
	if err != nil {
		return nil, internalServerError("Error getting user email from external provider").WithInternalError(err)
	}

	return userData, nil
}

func (a *API) OAuth2Provider(ctx context.Context, name string) (oauth2provider.OAuth2Provider, error) {
	providerCandidate, err := a.Provider(ctx, name)
	if err != nil {
		return nil, err
	}

	switch p := providerCandidate.(type) {
	case oauth2provider.OAuth2Provider:
		return p, nil
	default:
		return nil, badRequestError("Provider can not be used for OAuth")
	}
}
