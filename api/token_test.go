package api

import (
	"os"
	"testing"

	"github.com/gobuffalo/uuid"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TokenTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration

	token      string
	instanceID uuid.UUID
}

func TestToken(t *testing.T) {
	os.Setenv("GOTRUE_RATE_LIMIT_IP_LOOKUPS", "X-Forwarded-For")
	api, config, instanceID, err := setupAPIForTestForInstance()
	require.NoError(t, err)

	ts := &TokenTestSuite{
		API:        api,
		Config:     config,
		instanceID: instanceID,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *TokenTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)
}

// func (ts *TokenTestSuite) TestRateLimitToken() {
// 	var buffer bytes.Buffer
// 	req := httptest.NewRequest(http.MethodPost, "http://localhost/token", &buffer)
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("X-Forwarded-For", "1.2.3.4")

// 	for i := 0; i < 30; i++ {
// 		w := httptest.NewRecorder()
// 		ts.API.handler.ServeHTTP(w, req)
// 		assert.Equal(ts.T(), http.StatusBadRequest, w.Code)
// 	}
// 	w := httptest.NewRecorder()
// 	ts.API.handler.ServeHTTP(w, req)
// 	assert.Equal(ts.T(), http.StatusTooManyRequests, w.Code)
// }
