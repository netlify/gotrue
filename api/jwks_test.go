package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type JWKSTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	token      string
	instanceID uuid.UUID
}

func (ts *JWKSTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)
}

func TestJWKS(t *testing.T) {
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &JWKSTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}

	suite.Run(t, ts)
}

// TestJWKS tests API /.well-known/jwk.json route
func (ts *JWKSTestSuite) TestJWKS() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusOK, w.Code)
	respBody := string(w.Body.Bytes())
	var respJSON map[string]interface{}
	err := json.Unmarshal([]byte(respBody), &respJSON)
	require.NoError(ts.T(), err, "Failed to parse jwks response to JSON")

	keys, ok := respJSON["keys"].([]interface{})
	require.True(ts.T(), ok)
	require.Equal(ts.T(), 1, len(keys))

	firstKey, ok := keys[0].(map[string]interface{})
	require.True(ts.T(), ok)
	require.Equal(ts.T(), "RS256", firstKey["alg"])
	require.Equal(ts.T(), "AQAB", firstKey["e"])
	require.Equal(ts.T(), "IaVNE_1UoMhAtgBZ8raPvDuERW37uueMSUEPLD6lw60=", firstKey["kid"])
	require.Equal(ts.T(), "RSA", firstKey["kty"])
	require.Equal(ts.T(), "qPtmGv7HxJ9dgeZ8itz5ZrjHUemz-SvF-RbtfY8LRVyy66hZ1uSigs3dLHls5cKp1RoQqAoytjsGTBdKS1DLqLG7S4B0NL0YIKmLg_hqUqxJgwt60UfoVswj0pTSdunNW-39P20PiQWtOpJtVwztIN64zHnF5lJjI175k_37jpvAWoFmtSq5hWFeYw9ji1-aBarCJ-K9qog0mNZFeCko7LeKmtGxuodoYiUYVJ_q1c3su5FWs4ZdR-kv6GRvDMefc-LW92IuTsnCyGxZz_udxtJrpIg4ErenflIAg0VLDhCcAvvh4HuQ4yrIdp4pK9VaLh5ceNm4zKQ0oUcQN62LgUa_WK_Cwnvrw2O3-Ipm9a1ASHlejPf8s8a7LBCJ5jdSODgS4L3z28qY4gTeE_EB4f4ZuzCKBcgNlZ20pKV36gVeW2wCohSgFLWsNQQZeTYDyxaGXiLD8IRU6q533WIgdqoCfDU9Qr2qPnKre_tLr8MXBCLQpMWgkLC0rZn67LM2Ra_eGruY0MuFtG9jTbtP9SRa_TtdXQr53wnlLz_aZHJqxplJqkFXykLvnFSHq0iX6y6AnQ3gh0dL7ReDXQ3dHxvZdKJ3yQfVMnUdvJb5OWRUay3sxfuWwOYIMNDTSzuFTY7DveKX808G7DKmv9H78BX5UDtgCAgujBiTMYYPvSk", firstKey["n"])
	require.Equal(ts.T(), "sig", firstKey["use"])
	require.Equal(ts.T(), "36755eb4299e8bc21913a0f1889c7e38a404009e", firstKey["x5t"])
}
