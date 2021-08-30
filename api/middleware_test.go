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
	secret   string = "0x0000000000000000000000000000000000000000"
	response string = "10000000-aaaa-bbbb-cccc-000000000001"
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
		"email":          "test@example.com",
		"password":       "secret",
		"hcaptcha_token": response,
	}))

	ts.Config.Security.Captcha.Enabled = true
	ts.Config.Security.Captcha.Provider = "hcaptcha"
	ts.Config.Security.Captcha.Secret = secret

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
		"email":          "test@example.com",
		"password":       "secret",
		"hcaptcha_token": response,
	}))

	// check if body is the same
	require.Equal(ts.T(), body, buffer.Bytes())
	require.Equal(ts.T(), afterCtx, beforeCtx)
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
