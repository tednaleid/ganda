package parser_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/parser"
	"io"
	"strings"
	"testing"
)

func TestSendGetRequestUrlsHaveDefaultHeaders(t *testing.T) {
	t.Parallel()

	requestsWithContext := make(chan parser.RequestWithContext)
	defer close(requestsWithContext)

	firstLine := "https://example.com/bar"
	secondLine := "https://example.com/qux"
	var in = inputReader([]string{firstLine, secondLine})

	go parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})

	requestWithContext := <-requestsWithContext
	request := requestWithContext.Request
	requestContext := requestWithContext.RequestContext

	assert.Equal(t, firstLine, request.URL.String(), "expected url")
	assert.Equal(t, "GET", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")
	assert.Equal(t, []string{}, requestContext, "expected nil context")

	secondRequestWithContext := <-requestsWithContext
	secondRequest := secondRequestWithContext.Request
	secondRequestContext := secondRequestWithContext.RequestContext

	assert.Equal(t, secondLine, secondRequest.URL.String(), "expected url")
	assert.Equal(t, []string{}, secondRequestContext, "expected nil context")

}

func TestSendRequestsHasRaggedRequestContext(t *testing.T) {
	t.Parallel()

	requestsWithContext := make(chan parser.RequestWithContext, 3)
	defer close(requestsWithContext)

	// we allow ragged numbers of fields in the TSV input
	// we also follow the quoting rules in RFC 4180, so:
	//    a double quote in a quoted field is escaped with another double quote
	//    whitespace inside a quoted field is preserved
	inputLines := `
        https://ex.com/bar	foo	"quoted content"	
		https://ex.com/qux	quux	"  ""quoted with whitespace""  "	456
		https://ex.com/123	"baz"
    `

	var in = trimmedInputReader(inputLines)

	parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})

	expectedResults := []struct {
		url     string
		context []string
	}{
		{"https://ex.com/bar", []string{"foo", "quoted content"}},
		{"https://ex.com/qux", []string{"quux", "  \"quoted with whitespace\"  ", "456"}},
		{"https://ex.com/123", []string{"baz"}},
	}

	for _, expectedResult := range expectedResults {
		requestWithContext := <-requestsWithContext
		request := requestWithContext.Request
		requestContext := requestWithContext.RequestContext

		assert.Equal(t, expectedResult.url, request.URL.String(), "expected url")
		assert.Equal(t, expectedResult.context, requestContext, "expected context")
	}

}

func TestSendRequestsHasMalformedInput(t *testing.T) {
	t.Parallel()

	requestsWithContext := make(chan parser.RequestWithContext, 1)
	defer close(requestsWithContext)

	inputLines := `https://ex.com/bar	foo	"quoted content	missing terminating quote`

	var in = trimmedInputReader(inputLines)

	err := parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})

	assert.NotNil(t, err, "expected error")
	assert.Equal(t, err.Error(), "parse error on line 1, column 65: extraneous or missing \" in quoted-field")
}

func inputReader(urls []string) io.Reader {
	stringUrls := strings.Join(urls, "\n")
	return strings.NewReader(stringUrls)
}

func trimmedInputReader(s string) io.Reader {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.NewReader(strings.Join(lines, "\n"))
}
