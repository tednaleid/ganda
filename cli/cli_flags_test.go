package cli

import (
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"math"
	"strconv"
	"testing"
)

func TestHelp(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda", "-h"})
	assert.NotNil(t, results)
	assert.Nil(t, results.GetContext()) // context isn't set up when help is called
	assert.Equal(t, "", results.stderr) // help is not written to stderr when explicitly called
	assert.Contains(t, results.stdout, "NAME:\n   ganda")
}

func TestVersion(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda", "-v"})
	assert.NotNil(t, results)
	assert.Nil(t, results.GetContext()) // context isn't set up when version is called
	assert.Equal(t, "", results.stderr)
	assert.Equal(t, "ganda version "+testBuildInfo.ToString()+"\n", results.stdout)
}

func TestWorkers(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda", "-W", "10"})
	assert.NotNil(t, results)
	assert.Equal(t, 10, results.GetContext().RequestWorkers)
}

func TestRetries(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda"})
	assert.NotNil(t, results)
	assert.Equal(t, int64(0), results.GetContext().Retries)

	separateResults, _ := ParseGandaArgs([]string{"ganda", "--retry", "5"})
	assert.NotNil(t, separateResults)
	assert.Equal(t, int64(5), separateResults.GetContext().Retries)
}

func TestInvalidWorkers(t *testing.T) {
	testCases := []struct {
		input string
		error string
	}{
		{strconv.FormatInt(int64(math.MaxInt32)+1, 10), "value out of range"},
		{"foobar", "invalid value \"foobar\" for flag -W"},
	}

	for _, tc := range testCases {
		results, _ := ParseGandaArgs([]string{"ganda", "-W", tc.input})
		assert.NotNil(t, results)
		assert.Nil(t, results.GetContext())
		assert.Contains(t, results.stderr, tc.error)
	}
}

func TestResponseBodyFlags(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda"})
	assert.NotNil(t, results)
	assert.NotNil(t, results.GetContext())
	assert.Equal(t, config.Raw, results.GetContext().ResponseBody)

	testCases := []struct {
		input    string
		expected config.ResponseBodyType
	}{
		{"", config.Raw},
		{"base64", config.Base64},
		{"discard", config.Discard},
		{"escaped", config.Escaped},
		{"raw", config.Raw},
		{"sha256", config.Sha256},
	}

	for _, tc := range testCases {
		if tc.input == "" {
			results, _ := ParseGandaArgs([]string{"ganda"})
			assert.NotNil(t, results)
			assert.NotNil(t, results.GetContext())
			assert.Equal(t, tc.expected, results.GetContext().ResponseBody)
		} else {
			shortResults, _ := ParseGandaArgs([]string{"ganda", "-B", tc.input})
			assert.NotNil(t, shortResults)
			assert.NotNil(t, shortResults.GetContext())
			assert.Equal(t, tc.expected, shortResults.GetContext().ResponseBody)

			longResults, _ := ParseGandaArgs([]string{"ganda", "--response-body", tc.input})
			assert.NotNil(t, longResults)
			assert.NotNil(t, longResults.GetContext())
			assert.Equal(t, tc.expected, longResults.GetContext().ResponseBody)
		}
	}
}
