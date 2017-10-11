package api

import (
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HelpersTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration
}

func (ts *HelpersTestSuite) SetupTest() {
	api, config, err := NewAPIFromConfigFile("test.env", "v1")
	require.NoError(ts.T(), err)

	ts.API = api
	ts.Config = config
}

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}

func testPassword(a *API, password string) bool {
	_, isValid := a.isPasswordValid(password)
	return isValid
}

func (ts *HelpersTestSuite) TestHelpers_IsPasswordValid() {
	assert.False(ts.T(), testPassword(ts.API, ""))
	assert.True(ts.T(), testPassword(ts.API, "1"))
	ts.API.config.Password.MinLength = 6
	assert.False(ts.T(), testPassword(ts.API, "fail"))
	assert.True(ts.T(), testPassword(ts.API, "passwd"))
	ts.API.config.Password.MinNumbers = 2
	assert.False(ts.T(), testPassword(ts.API, "passwd"))
	assert.True(ts.T(), testPassword(ts.API, "1passwd2"))
	ts.API.config.Password.MinSymbols = 3
	assert.False(ts.T(), testPassword(ts.API, "1passwd2"))
	assert.False(ts.T(), testPassword(ts.API, "1pa#ss%wd2"))
	assert.True(ts.T(), testPassword(ts.API, "1pa#ss%wd2@"))
	ts.API.config.Password.MinUppercase = 2
	assert.False(ts.T(), testPassword(ts.API, "1Passwd2"))
	assert.True(ts.T(), testPassword(ts.API, "1Pa#s%Wd2@"))
}
