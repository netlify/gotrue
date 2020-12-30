package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gobuffalo/uuid"

	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var testUUID = uuid.Must(uuid.FromString("11111111-1111-1111-1111-111111111111"))

const operatorToken = "operatorToken"

type InstanceTestSuite struct {
	suite.Suite
	API *API
}

func TestInstance(t *testing.T) {
	api, _, err := setupAPIForMultiinstanceTest()
	require.NoError(t, err)

	api.config.OperatorToken = operatorToken

	ts := &InstanceTestSuite{
		API: api,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *InstanceTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)
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
	req := httptest.NewRequest(http.MethodPost, "/instances", &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	// Setup response recorder
	w := httptest.NewRecorder()

	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), http.StatusCreated, w.Code)
	resp := models.Instance{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(ts.T(), resp.BaseConfig)

	i, err := models.GetInstanceByUUID(ts.API.db, testUUID)
	require.NoError(ts.T(), err)
	assert.NotNil(ts.T(), i.BaseConfig)
}

func (ts *InstanceTestSuite) TestGet() {
	instanceID := uuid.Must(uuid.NewV4())
	err := ts.API.db.Create(&models.Instance{
		ID:   instanceID,
		UUID: testUUID,
		BaseConfig: &conf.Configuration{
			JWT: conf.JWTConfiguration{
				Secret: "testsecret",
			},
		},
	})
	require.NoError(ts.T(), err)

	req := httptest.NewRequest(http.MethodGet, "/instances/"+instanceID.String(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)
	resp := models.Instance{}
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&resp))
}

func (ts *InstanceTestSuite) TestUpdate() {
	instanceID := uuid.Must(uuid.NewV4())
	err := ts.API.db.Create(&models.Instance{
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

	req := httptest.NewRequest(http.MethodPut, "/instances/"+instanceID.String(), &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)

	i, err := models.GetInstanceByUUID(ts.API.db, testUUID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), i.BaseConfig.JWT.Secret, "testsecret")
	require.Equal(ts.T(), i.BaseConfig.SiteURL, "https://test.mysite.com")
}

func (ts *InstanceTestSuite) TestUpdate_DisableEmail() {
	instanceID := uuid.Must(uuid.NewV4())
	err := ts.API.db.Create(&models.Instance{
		ID:   instanceID,
		UUID: testUUID,
		BaseConfig: &conf.Configuration{
			External: conf.ProviderConfiguration{
				Email: conf.EmailProviderConfiguration{
					Disabled: false,
				},
			},
		},
	})
	require.NoError(ts.T(), err)

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"config": &conf.Configuration{
			External: conf.ProviderConfiguration{
				Email: conf.EmailProviderConfiguration{
					Disabled: true,
				},
			},
		},
	}))

	req := httptest.NewRequest(http.MethodPut, "/instances/"+instanceID.String(), &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)

	i, err := models.GetInstanceByUUID(ts.API.db, testUUID)
	require.NoError(ts.T(), err)
	require.True(ts.T(), i.BaseConfig.External.Email.Disabled)
}

func (ts *InstanceTestSuite) TestUpdate_PreserveSMTPConfig() {
	instanceID := uuid.Must(uuid.NewV4())
	err := ts.API.db.Create(&models.Instance{
		ID:   instanceID,
		UUID: testUUID,
		BaseConfig: &conf.Configuration{
			SMTP: conf.SMTPConfiguration{
				Host: "foo.com",
				User: "Admin",
				Pass: "password123",
			},
		},
	})
	require.NoError(ts.T(), err)

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"config": &conf.Configuration{
			Mailer: conf.MailerConfiguration{
				Subjects:  conf.EmailContentConfiguration{Invite: "foo"},
				Templates: conf.EmailContentConfiguration{Invite: "bar"},
			},
		},
	}))

	req := httptest.NewRequest(http.MethodPut, "/instances/"+instanceID.String(), &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)

	i, err := models.GetInstanceByUUID(ts.API.db, testUUID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), "password123", i.BaseConfig.SMTP.Pass)
}

func (ts *InstanceTestSuite) TestUpdate_ClearPassword() {
	instanceID := uuid.Must(uuid.NewV4())
	err := ts.API.db.Create(&models.Instance{
		ID:   instanceID,
		UUID: testUUID,
		BaseConfig: &conf.Configuration{
			SMTP: conf.SMTPConfiguration{
				Host: "foo.com",
				User: "Admin",
				Pass: "password123",
			},
		},
	})
	require.NoError(ts.T(), err)

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"config": map[string]interface{}{
			"smtp": map[string]interface{}{
				"pass": "",
			},
		},
	}))
	ts.T().Log(buffer.String())

	req := httptest.NewRequest(http.MethodPut, "/instances/"+instanceID.String(), &buffer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+operatorToken)

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)
	require.Equal(ts.T(), w.Code, http.StatusOK)

	i, err := models.GetInstanceByUUID(ts.API.db, testUUID)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), "", i.BaseConfig.SMTP.Pass)
}
