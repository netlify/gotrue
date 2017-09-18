package api

import (
	"testing"

	"github.com/netlify/gotrue/conf"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HelpersTestSuite struct {
	suite.Suite
	API    *API
	Config *conf.Configuration
}

func (ts *HelpersTestSuite) SetupSuite() {
}

func (ts *HelpersTestSuite) TearDownSuite() {
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

func (ts *HelpersTestSuite) TestHelpers_IsPasswordValid() {
	require.False(ts.T(), ts.API.isPasswordValid(""), "password should not be valid")
	require.True(ts.T(), ts.API.isPasswordValid("1"), "password should be valid")
	ts.API.config.Password.MinLength = 6
	require.False(ts.T(), ts.API.isPasswordValid("fail"), "password should not be valid")
	require.True(ts.T(), ts.API.isPasswordValid("passwd"), "password should be valid")
	ts.API.config.Password.MinNumbers = 2
	require.False(ts.T(), ts.API.isPasswordValid("passwd"), "password should not be valid")
	require.True(ts.T(), ts.API.isPasswordValid("1passwd2"), "password should be valid")
	ts.API.config.Password.MinSymbols = 3
	require.False(ts.T(), ts.API.isPasswordValid("1passwd2"), "password should not be valid")
	require.False(ts.T(), ts.API.isPasswordValid("1pa#ss%wd2"), "password should not be valid")
	require.True(ts.T(), ts.API.isPasswordValid("1pa#ss%wd2@"), "password should be valid")
	ts.API.config.Password.MinUppercase = 2
	require.False(ts.T(), ts.API.isPasswordValid("1Passwd2"), "password should not be valid")
	require.True(ts.T(), ts.API.isPasswordValid("1Pa#s%Wd2@"), "password should be be valid")
}
