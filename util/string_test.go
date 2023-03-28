package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StringTestSuite struct {
	suite.Suite
}

func (suite *StringTestSuite) SetupSuite() {
}

func (suite *StringTestSuite) SetupTest() {
}

func (suite *StringTestSuite) TearDownTest() {
}

func (suite *StringTestSuite) TearDownSuite() {
}

func (suite *StringTestSuite) TestGetStringInBetween() {
	require := suite.Require()
	testCates := []struct {
		str      string
		startStr string
		endStr   string
		expected string
	}{
		{`"test"`, `"`, `"`, `test`},
		{`'test'`, `'`, `'`, `test`},
		{"`test`", "`", "`", `test`},
		{`.test.`, `.`, `.`, `test`},
		{"\ntest\n", "\n", "\n", `test`},
		{`some-text-test-some-text`, `some-text-`, `-some-text`, `test`},
	}

	for _, tc := range testCates {
		require.Equal(tc.expected, GetStringInBetween(tc.str, tc.startStr, tc.endStr))
	}
}

func TestString(t *testing.T) {
	suite.Run(t, new(StringTestSuite))
}
