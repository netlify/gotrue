package api

import (
	"encoding/json"
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
	req := httptest.NewRequest("GET", "http://localhost/", nil)
	w := httptest.NewRecorder()
	ctx := req.Context()
	ts.API.Index(ctx, w, req)

	resp := w.Result()
	assert.Equal(ts.T(), resp.StatusCode, 200)

	// Check response data
	data := make(map[string]string)
	require.NoError(ts.T(), json.NewDecoder(resp.Body).Decode(&data))

	assert.Equal(ts.T(), data["name"], "GoTrue")
	assert.Equal(ts.T(), data["version"], "v1")
}

func TestIndex(t *testing.T) {
	suite.Run(t, new(IndexTestSuite))
}
