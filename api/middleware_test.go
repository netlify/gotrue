package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	HCaptchaSecret   string = "0x0000000000000000000000000000000000000000"
	HCaptchaResponse string = "10000000-aaaa-bbbb-cccc-000000000001"
)

type MiddlewareTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	instanceID uuid.UUID
}

func TestHCaptcha(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &MiddlewareTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *MiddlewareTestSuite) TestVerifyCaptchaValid() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test@example.com",
		"password": "secret",
		"gotrue_meta_security": map[string]interface{}{
			"hcaptcha_token": HCaptchaResponse,
		},
	}))

	ts.Config.Security.Captcha.Enabled = true
	ts.Config.Security.Captcha.Provider = "hcaptcha"
	ts.Config.Security.Captcha.Secret = HCaptchaSecret

	req := httptest.NewRequest(http.MethodPost, "http://localhost", &buffer)
	req.Header.Set("Content-Type", "application/json")
	beforeCtx, err := WithInstanceConfig(req.Context(), ts.Config, ts.instanceID)
	require.NoError(ts.T(), err)

	req = req.WithContext(beforeCtx)

	w := httptest.NewRecorder()

	afterCtx, err := ts.API.verifyCaptcha(w, req)
	require.NoError(ts.T(), err)

	body, err := ioutil.ReadAll(req.Body)

	// re-initialize buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"email":    "test@example.com",
		"password": "secret",
		"gotrue_meta_security": map[string]interface{}{
			"hcaptcha_token": HCaptchaResponse,
		},
	}))

	// check if body is the same
	require.Equal(ts.T(), body, buffer.Bytes())
	require.Equal(ts.T(), afterCtx, beforeCtx)
}

func (ts *MiddlewareTestSuite) TestVerifyCaptchaInvalid() {
	cases := []struct {
		desc         string
		captchaConf  *conf.CaptchaConfiguration
		expectedCode int
		expectedMsg  string
	}{
		{
			"Unsupported provider",
			&conf.CaptchaConfiguration{
				Enabled:  true,
				Provider: "test",
			},
			http.StatusInternalServerError,
			"server misconfigured",
		},
		{
			"Missing secret",
			&conf.CaptchaConfiguration{
				Enabled:  true,
				Provider: "hcaptcha",
				Secret:   "",
			},
			http.StatusInternalServerError,
			"server misconfigured",
		},
		{
			"Captcha validation failed",
			&conf.CaptchaConfiguration{
				Enabled:  true,
				Provider: "hcaptcha",
				Secret:   "test",
			},
			http.StatusInternalServerError,
			"request validation failure",
		},
	}
	for _, c := range cases {
		ts.Run(c.desc, func() {
			ts.Config.Security.Captcha = *c.captchaConf
			var buffer bytes.Buffer
			require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
				"email":    "test@example.com",
				"password": "secret",
				"gotrue_meta_security": map[string]interface{}{
					"hcaptcha_token": HCaptchaResponse,
				},
			}))
			req := httptest.NewRequest(http.MethodPost, "http://localhost", &buffer)
			req.Header.Set("Content-Type", "application/json")
			ctx, err := WithInstanceConfig(req.Context(), ts.Config, ts.instanceID)
			require.NoError(ts.T(), err)

			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			_, err = ts.API.verifyCaptcha(w, req)
			require.Equal(ts.T(), c.expectedCode, err.(*HTTPError).Code)
			require.Equal(ts.T(), c.expectedMsg, err.(*HTTPError).Message)
		})
	}
}

func TestFunctionHooksUnmarshalJSON(t *testing.T) {
	tests := []struct {
		in string
		ok bool
	}{
		{`{ "signup" : "identity-signup" }`, true},
		{`{ "signup" : ["identity-signup"] }`, true},
		{`{ "signup" : {"foo" : "bar"} }`, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			var f FunctionHooks
			err := json.Unmarshal([]byte(tt.in), &f)
			if tt.ok {
				assert.NoError(t, err)
				assert.Equal(t, FunctionHooks{"signup": {"identity-signup"}}, f)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
