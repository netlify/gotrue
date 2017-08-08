package api

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/netlify/gotrue/models"
)

const (
	instanceIDHeaderName          = "x-nf-id"
	instanceEnvironmentHeaderName = "x-nf-env"
	siteURLHeaderName             = "x-nf-site-url"
)

func addGetBody(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	if req.Method == http.MethodGet {
		return req.Context(), nil
	}

	if req.Body == nil || req.Body == http.NoBody {
		return nil, badRequestError("request must provide a body")
	}

	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, internalServerError("Error reading body").WithInternalError(err)
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(buf)), nil
	}
	req.Body, _ = req.GetBody()
	return req.Context(), nil
}

func (api *API) loadInstanceConfig(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()

	instanceID := r.Header.Get(instanceIDHeaderName)
	if instanceID == "" {
		return nil, badRequestError("Netlify microservice headers missing")
	}

	// TODO verify JWS of microservice API request

	logEntrySetField(r, "instance_id", instanceID)
	instance, err := api.db.GetInstance(instanceID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError("Unable to locate site configuration")
		}
		return nil, internalServerError("Database error loading instance").WithInternalError(err)
	}

	env := r.Header.Get(instanceEnvironmentHeaderName)
	if env == "" {
		return nil, badRequestError("No environment specified")
	}
	logEntrySetField(r, "env", env)
	config, err := instance.ConfigForEnvironment(env)
	if err != nil {
		return nil, internalServerError("Error loading environment config").WithInternalError(err)
	}

	if siteURL := r.Header.Get(siteURLHeaderName); siteURL != "" {
		config.SiteURL = siteURL
	}
	logEntrySetField(r, "site_url", config.SiteURL)

	ctx, err = WithInstanceConfig(ctx, config, instanceID)
	if err != nil {
		return nil, internalServerError("Error loading instance config").WithInternalError(err)
	}

	return ctx, nil
}

func verifyNetlifyRequest(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	// TODO verify microservice management API request came from Netlify
	return req.Context(), nil
}
