package api

import (
	"context"
	"net/http"

	"github.com/netlify/gotrue/api/provider"
)

func (a *API) loadSAMLState(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	state := r.FormValue("RelayState")
	if state == "" {
		return nil, badRequestError("SAML RelayState is missing")
	}

	ctx := r.Context()

	return a.loadExternalState(ctx, state)
}

func (a *API) samlCallback(r *http.Request, ctx context.Context) (*provider.UserProvidedData, error) {
	config := a.getConfig(ctx)

	samlProvider, err := provider.NewSamlProvider(config.External.Saml, a.db, getInstanceID(ctx))
	if err != nil {
		return nil, badRequestError("Could not initialize SAML provider: %+v", err).WithInternalError(err)
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		return nil, badRequestError("SAML Response is missing")
	}

	assertionInfo, err := samlProvider.ServiceProvider.RetrieveAssertionInfo(samlResponse)
	if err != nil {
		return nil, internalServerError("Parsing SAML assertion failed: %+v", err).WithInternalError(err)
	}

	if assertionInfo.WarningInfo.InvalidTime {
		return nil, forbiddenError("SAML response has invalid time")
	}

	if assertionInfo.WarningInfo.NotInAudience {
		return nil, forbiddenError("SAML response is not in audience")
	}

	if assertionInfo == nil {
		return nil, internalServerError("SAML Assertion is missing")
	}
	userData := &provider.UserProvidedData{
		Emails: []provider.Email{{
			Email:    assertionInfo.NameID,
			Verified: true,
		}},
	}
	return userData, nil
}

func (a *API) SAMLMetadata(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	config := getConfig(ctx)

	samlProvider, err := provider.NewSamlProvider(config.External.Saml, a.db, getInstanceID(ctx))
	if err != nil {
		return internalServerError("Could not create SAML Provider: %+v", err).WithInternalError(err)
	}

	metadata, err := samlProvider.SPMetadata()
	w.Header().Set("Content-Type", "application/xml")
	w.Write(metadata)
	return nil
}
