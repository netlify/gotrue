package api

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/gotrue/models"
)

const (
	jwsSignatureHeaderName = "x-nf-sign"
)

type NetlifyMicroserviceClaims struct {
	SiteURL    string `json:"site_url"`
	Env        string `json:"env"`
	InstanceID string `json:"id"`
	NetlifyID  string `json:"netlify_id"`
	jwt.StandardClaims
}

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

	signature := r.Header.Get(jwsSignatureHeaderName)
	if signature == "" {
		return nil, badRequestError("Netlify microservice headers missing")
	}

	claims := NetlifyMicroserviceClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err := p.ParseWithClaims(signature, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(api.config.NetlifySecret), nil
	})
	if err != nil {
		return nil, badRequestError("Netlify microservice headers are invalid: %v", err)
	}

	instanceID := claims.InstanceID
	if instanceID == "" {
		return nil, badRequestError("Netlify microservice headers missing")
	}

	logEntrySetField(r, "instance_id", instanceID)
	instance, err := api.db.GetInstance(instanceID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError("Unable to locate site configuration")
		}
		return nil, internalServerError("Database error loading instance").WithInternalError(err)
	}

	env := claims.Env
	if env == "" {
		return nil, badRequestError("No environment specified")
	}
	logEntrySetField(r, "env", env)
	config, err := instance.ConfigForEnvironment(env)
	if err != nil {
		return nil, internalServerError("Error loading environment config").WithInternalError(err)
	}

	if claims.SiteURL != "" {
		config.SiteURL = claims.SiteURL
	}
	logEntrySetField(r, "site_url", config.SiteURL)

	ctx, err = WithInstanceConfig(ctx, config, instanceID)
	if err != nil {
		return nil, internalServerError("Error loading instance config").WithInternalError(err)
	}

	return ctx, nil
}

func (api *API) verifyNetlifyRequest(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	token := strings.TrimPrefix(req.Header.Get("authorization"), "Bearer ")
	if token != api.config.NetlifySecret {
		return nil, unauthorizedError("Request did not originate from Netlify")
	}
	return req.Context(), nil
}
