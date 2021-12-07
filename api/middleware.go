package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/didip/tollbooth/v5"
	"github.com/didip/tollbooth/v5/limiter"
	"github.com/gobuffalo/uuid"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/netlify/gotrue/models"
)

const (
	jwsSignatureHeaderName = "x-nf-sign"
)

type FunctionHooks map[string][]string

type NetlifyMicroserviceClaims struct {
	jwt.StandardClaims
	SiteURL       string        `json:"site_url"`
	InstanceID    string        `json:"id"`
	NetlifyID     string        `json:"netlify_id"`
	FunctionHooks FunctionHooks `json:"function_hooks"`
}

func (f *FunctionHooks) UnmarshalJSON(b []byte) error {
	var raw map[string][]string
	err := json.Unmarshal(b, &raw)
	if err == nil {
		*f = FunctionHooks(raw)
		return nil
	}
	// If unmarshaling into map[string][]string fails, try legacy format.
	var legacy map[string]string
	err = json.Unmarshal(b, &legacy)
	if err != nil {
		return err
	}
	if *f == nil {
		*f = make(FunctionHooks)
	}
	for event, hook := range legacy {
		(*f)[event] = []string{hook}
	}
	return nil
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

func (a *API) loadJWSSignatureHeader(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()
	signature := r.Header.Get(jwsSignatureHeaderName)
	if signature == "" {
		return nil, badRequestError("Operator microservice headers missing")
	}
	return withSignature(ctx, signature), nil
}

func (a *API) loadInstanceConfig(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()

	signature := getSignature(ctx)
	if signature == "" {
		return nil, badRequestError("Operator signature missing")
	}

	claims := NetlifyMicroserviceClaims{}
	p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
	_, err := p.ParseWithClaims(signature, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.OperatorToken), nil
	})
	if err != nil {
		return nil, badRequestError("Operator microservice signature is invalid: %v", err)
	}

	if claims.InstanceID == "" {
		return nil, badRequestError("Instance ID is missing")
	}
	instanceID, err := uuid.FromString(claims.InstanceID)
	if err != nil {
		return nil, badRequestError("Instance ID is not a valid UUID")
	}

	logEntrySetField(r, "instance_id", instanceID)
	logEntrySetField(r, "netlify_id", claims.NetlifyID)
	instance, err := models.GetInstance(a.db, instanceID)
	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError("Unable to locate site configuration")
		}
		return nil, internalServerError("Database error loading instance").WithInternalError(err)
	}

	config, err := instance.Config()
	if err != nil {
		return nil, internalServerError("Error loading environment config").WithInternalError(err)
	}

	if claims.SiteURL != "" {
		config.SiteURL = claims.SiteURL
	}
	logEntrySetField(r, "site_url", config.SiteURL)

	ctx = withNetlifyID(ctx, claims.NetlifyID)
	ctx = withFunctionHooks(ctx, claims.FunctionHooks)

	ctx, err = WithInstanceConfig(ctx, config, instanceID)
	if err != nil {
		return nil, internalServerError("Error loading instance config").WithInternalError(err)
	}

	return ctx, nil
}

func (a *API) limitHandler(lmt *limiter.Limiter) middlewareHandler {
	return func(w http.ResponseWriter, req *http.Request) (context.Context, error) {
		c := req.Context()
		if limitHeader := a.config.RateLimitHeader; limitHeader != "" {
			key := req.Header.Get(a.config.RateLimitHeader)
			err := tollbooth.LimitByKeys(lmt, []string{key})
			if err != nil {
				return c, httpError(http.StatusTooManyRequests, "Rate limit exceeded")
			}
		}
		return c, nil
	}
}

func (a *API) verifyOperatorRequest(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	c, _, err := a.extractOperatorRequest(w, req)
	return c, err
}

func (a *API) extractOperatorRequest(w http.ResponseWriter, req *http.Request) (context.Context, string, error) {
	token, err := a.extractBearerToken(w, req)
	if err != nil {
		return nil, token, err
	}
	if token == "" || token != a.config.OperatorToken {
		return nil, token, unauthorizedError("Request does not include an Operator token")
	}
	return withAdminUser(req.Context(), &models.User{ID: uuid.Nil, Email: "operator@netlify.com"}), token, nil
}

func (a *API) requireAdminCredentials(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	c, t, err := a.extractOperatorRequest(w, req)
	if err == nil {
		return c, nil
	}

	if t == "" {
		return nil, err
	}

	c, err = a.parseJWTClaims(t, req, w)
	if err != nil {
		return nil, err
	}

	return a.requireAdmin(c, w, req)
}

func (a *API) requireEmailProvider(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	ctx := req.Context()
	config := a.getConfig(ctx)

	if config.External.Email.Disabled {
		return nil, badRequestError("Unsupported email provider")
	}

	return ctx, nil
}
