package cli

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/execcontext"
	"math"
	"strconv"
	"strings"
	"testing"
)

var testBuildInfo = BuildInfo{Version: "testing", Commit: "123abc", Date: "2023-12-20"}

type runResults struct {
	stderr  string
	stdout  string
	context *execcontext.Context
}

func parseArgs(args []string) runResults {
	in := strings.NewReader("")
	err := new(bytes.Buffer)
	out := new(bytes.Buffer)
	var resultContext *execcontext.Context

	processRequests := func(context *execcontext.Context) {
		resultContext = context
	}

	command := SetupCommand(testBuildInfo, in, err, out, processRequests)
	command.Run(context.Background(), args)
	return runResults{err.String(), out.String(), resultContext}
}

func TestHelp(t *testing.T) {
	results := parseArgs([]string{"ganda", "-h"})
	assert.NotNil(t, results)
	assert.Nil(t, results.context)      // context isn't set up when help is called
	assert.Equal(t, "", results.stderr) // help is not written to stderr when explicitly called
	assert.Contains(t, results.stdout, "NAME:\n   ganda")
}

func TestVersion(t *testing.T) {
	results := parseArgs([]string{"ganda", "-v"})
	assert.NotNil(t, results)
	assert.Nil(t, results.context) // context isn't set up when version is called
	assert.Equal(t, "", results.stderr)
	assert.Equal(t, "ganda version "+testBuildInfo.ToString()+"\n", results.stdout)
}

func TestWorkers(t *testing.T) {
	results := parseArgs([]string{"ganda", "-W", "10"})
	assert.NotNil(t, results)
	assert.Equal(t, 10, results.context.RequestWorkers)
	assert.Equal(t, 10, results.context.ResponseWorkers)

	separateResults := parseArgs([]string{"ganda", "-W", "10", "--response-workers", "5"})
	assert.NotNil(t, results)
	assert.Equal(t, 10, separateResults.context.RequestWorkers)
	assert.Equal(t, 5, separateResults.context.ResponseWorkers)
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
		results := parseArgs([]string{"ganda", "-W", tc.input})
		assert.NotNil(t, results)
		assert.Nil(t, results.context)
		assert.Contains(t, results.stderr, tc.error)
	}
}
