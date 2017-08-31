package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pborman/uuid"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const testUUID = "11111111-1111-1111-1111-111111111111"
const operatorToken = "operatorToken"

type InstanceTestSuite struct {
	suite.Suite
	API *API
}

func (ts *InstanceTestSuite) SetupSuite() {
	require.NoError(ts.T(), os.Setenv("GOTRUE_DB_DATABASE_URL", createTestDB()))
}

func (ts *InstanceTestSuite) TearDownSuite() {
	os.Remove(ts.API.config.DB.URL)
}

func (ts *InstanceTestSuite) SetupTest() {
	globalConfig, err := conf.LoadGlobal("test.env")
	require.NoError(ts.T(), err)
	globalConfig.OperatorToken = operatorToken
	globalConfig.MultiInstanceMode = true
	db, err := dial.Dial(globalConfig)
	require.NoError(ts.T(), err)

	api := NewAPI(globalConfig, db)
	ts.API = api

	// Cleanup existing user
	i, err := ts.API.db.GetInstanceByUUID(testUUID)
	if err == nil {
		require.NoError(ts.T(), api.db.DeleteInstance(i))
	}
}

func (ts *InstanceTestSuite) TestCreate() {
	// Request body
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"uuid":     testUUID,
		"site_url": "https://example.netlify.com",
		"config": map[string]interface{}{
			"jwt": map[string]interface{}{
				"secret": "testsecret",
			},
		},
	}))

	// Setup request
	req := httptest.NewRequest(http.MethodPost, "http://localhost/instances", &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusCreated)
	resp := models.Instance{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(ts.T(), resp.BaseConfig)

	i, err := ts.API.db.GetInstanceByUUID(testUUID)
	require.NoError(ts.T(), err)
	assert.NotNil(ts.T(), i.BaseConfig)
}

func (ts *InstanceTestSuite) TestGet() {
	instanceID := uuid.NewRandom().String()
	err := ts.API.db.CreateInstance(&models.Instance{
		ID:   instanceID,
		UUID: testUUID,
		BaseConfig: &conf.Configuration{
			JWT: conf.JWTConfiguration{
				Secret: "testsecret",
			},
		},
	})
	require.NoError(ts.T(), err)

	req := httptest.NewRequest(http.MethodGet, "http://localhost/instances/"+instanceID, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)
	resp := models.Instance{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&resp))
}

func (ts *InstanceTestSuite) TestUpdate() {
	instanceID := uuid.NewRandom().String()
	err := ts.API.db.CreateInstance(&models.Instance{
		ID:   instanceID,
		UUID: testUUID,
		BaseConfig: &conf.Configuration{
			JWT: conf.JWTConfiguration{
				Secret: "testsecret",
			},
		},
	})
	require.NoError(ts.T(), err)

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"config": &conf.Configuration{
			JWT: conf.JWTConfiguration{
				Secret: "testsecret",
			},
			SiteURL: "https://test.mysite.com",
		},
	}))

	req := httptest.NewRequest(http.MethodPut, "http://localhost/instances/"+instanceID, &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)

	i, err := ts.API.db.GetInstanceByUUID(testUUID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), i.BaseConfig.JWT.Secret, "testsecret")
	require.Equal(ts.T(), i.BaseConfig.SiteURL, "https://test.mysite.com")
}

func TestInstance(t *testing.T) {
	suite.Run(t, new(InstanceTestSuite))
}
