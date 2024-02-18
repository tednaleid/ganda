package cli

import (
	"bytes"
	ctx "context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/urfave/cli/v3"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// test helper structs and functions, no actual tests

var testBuildInfo = BuildInfo{Version: "testing", Commit: "123abc", Date: "2023-12-20"}

type GandaResults struct {
	stderr  string
	stdout  string
	command *cli.Command
}

func (results *GandaResults) GetContext() *execcontext.Context {
	context := results.command.Metadata["context"]
	if context == nil {
		return nil
	}
	return context.(*execcontext.Context)
}

func (results *GandaResults) assert(t *testing.T, expectedStandardOut string, expectedLog string) {
	assert.Equal(t, expectedStandardOut, results.stdout, "expected stdout")
	assert.Equal(t, expectedLog, results.stderr, "expected logger stderr")
}

// we want to test parsing of arguments, we don't actually want to execute any requests
func ParseGandaArgs(args []string) (GandaResults, error) {
	in := strings.NewReader("")
	return RunGanda(args, in)
}

func RunGanda(args []string, in io.Reader) (GandaResults, error) {
	stderr := new(bytes.Buffer)
	stdout := new(bytes.Buffer)

	command := SetupCommand(testBuildInfo, in, stderr, stdout)

	err := command.Run(ctx.Background(), args)

	return GandaResults{stderr.String(), stdout.String(), command}, err
}

type HttpServerStub struct {
	*httptest.Server
}

// The passed in handler function can verify the request and write a response given that input
func NewHttpServerStub(handler http.Handler) *HttpServerStub {
	return &HttpServerStub{httptest.NewServer(handler)}
}

// append the fragment to the end of the server base url
func (server *HttpServerStub) urlFor(fragment string) string {
	return fmt.Sprintf("%s/%s", server.URL, fragment)
}

func (server *HttpServerStub) urlsFor(fragments []string) []string {
	urls := make([]string, len(fragments))
	for i, path := range fragments {
		urls[i] = server.urlFor(path)
	}
	return urls
}

// stub stdin for the path fragment to create an url for this server
func (server *HttpServerStub) stubStdinUrl(fragment string) io.Reader {
	return server.stubStdinUrls([]string{fragment})
}

// given an array of paths, we will create a stub of stdin that has one url per line for our server stub
func (server *HttpServerStub) stubStdinUrls(fragments []string) io.Reader {
	urls := server.urlsFor(fragments)
	urlsString := strings.Join(urls, "\n")
	return strings.NewReader(urlsString)
}

func trimmedInputReader(s string) io.Reader {
	return strings.NewReader(trimIndent(s))
}

func trimIndentKeepTrailingNewline(s string) string {
	return trimIndent(s) + "\n"
}

func trimIndent(s string) string {
	lines := strings.Split(s, "\n")
	var trimmedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 {
			trimmedLines = append(trimmedLines, trimmedLine)
		}
	}
	return strings.Join(trimmedLines, "\n")
}
