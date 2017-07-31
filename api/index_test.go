package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IndexTestSuite struct {
	suite.Suite
	API *API
}

func (ts *IndexTestSuite) SetupTest() {
	api, err := NewAPIFromConfigFile("config.test.json", "v1")
	require.NoError(ts.T(), err)
	ts.API = api
}

// TestIndex tests API / route
func (ts *IndexTestSuite) TestIndex() {
	// Setup request and response reader
	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), w.Code, http.StatusOK)

	// Check response data
	data := make(map[string]string)
	require.NoError(ts.T(), json.NewDecoder(w.Body).Decode(&data))

	assert.Equal(ts.T(), data["name"], "GoTrue")
	assert.Equal(ts.T(), data["version"], "v1")
}

func TestIndex(t *testing.T) {
	suite.Run(t, new(IndexTestSuite))
}
