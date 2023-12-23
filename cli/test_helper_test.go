package cli

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/execcontext"
	"io"
	"strings"
	"testing"
)

var testBuildInfo = BuildInfo{Version: "testing", Commit: "123abc", Date: "2023-12-20"}

type RunResults struct {
	stderr  string
	stdout  string
	context *execcontext.Context
}

func (results RunResults) assert(t *testing.T, expectedStandardOut string, expectedLog string) {
	assert.Equal(t, expectedStandardOut, results.stdout, "expected stdout")
	assert.Equal(t, expectedLog, results.stderr, "expected logger stderr")
}

// we want to test parsing of arguments, we don't actually want to execute any requests
func ParseArgs(args []string) (RunResults, error) {
	in := strings.NewReader("")
	return runApp(args, in, nil)
}

// we want to control what stdin is sending and actually send the requests through
func RunApp(args []string, in io.Reader) (RunResults, error) {
	return runApp(args, in, ProcessRequests)
}

func runApp(args []string, in io.Reader, runBlock func(context *execcontext.Context)) (RunResults, error) {
	var resultContext *execcontext.Context
	stderr := new(bytes.Buffer)
	stdout := new(bytes.Buffer)

	processRequests := func(context *execcontext.Context) {
		resultContext = context
		if runBlock != nil {
			runBlock(context)
		}
	}

	err := RunCommand(testBuildInfo, args, in, stderr, stdout, processRequests)
	return RunResults{stderr.String(), stdout.String(), resultContext}, err
}
