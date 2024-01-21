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
	requestsWithContext := make(chan parser.RequestWithContext, 2)
	defer close(requestsWithContext)

	inputLines := `
        https://example.com/bar
        https://example.com/qux
    `

	var in = trimmedInputReader(inputLines)

	err := parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})

	assert.Nil(t, err, "expected no error")

	requestWithContext := <-requestsWithContext
	request := requestWithContext.Request
	requestContext := requestWithContext.RequestContext

	assert.Equal(t, "https://example.com/bar", request.URL.String(), "expected url")
	assert.Equal(t, "GET", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")
	assert.Equal(t, []string{}, requestContext, "expected nil context")

	secondRequestWithContext := <-requestsWithContext
	secondRequest := secondRequestWithContext.Request
	secondRequestContext := secondRequestWithContext.RequestContext

	assert.Equal(t, "https://example.com/qux", secondRequest.URL.String(), "expected url")
	assert.Equal(t, []string{}, secondRequestContext, "expected nil context")

}

func TestSendGetRequestUrlsAddGivenHeaders(t *testing.T) {
	requestsWithContext := make(chan parser.RequestWithContext, 1)
	defer close(requestsWithContext)

	inputLines := `https://example.com/bar`
	var in = trimmedInputReader(inputLines)

	requestHeaders := []config.RequestHeader{{Key: "X-Test", Value: "foo"}, {Key: "X-Test2", Value: "bar"}}

	err := parser.SendRequests(requestsWithContext, in, "GET", requestHeaders)

	assert.Nil(t, err, "expected no error")

	requestWithContext := <-requestsWithContext
	request := requestWithContext.Request
	requestContext := requestWithContext.RequestContext

	assert.Equal(t, "https://example.com/bar", request.URL.String(), "expected url")
	assert.Equal(t, "GET", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")
	assert.Equal(t, request.Header["X-Test"][0], "foo")
	assert.Equal(t, request.Header["X-Test2"][0], "bar")
	assert.Equal(t, []string{}, requestContext, "expected nil context")
}

func TestSendRequestsHasRaggedRequestContext(t *testing.T) {
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

func TestSendRequestsHasMalformedTSV(t *testing.T) {
	requestsWithContext := make(chan parser.RequestWithContext, 1)
	defer close(requestsWithContext)

	inputLines := `https://ex.com/bar	foo	"quoted content	missing terminating quote`

	var in = trimmedInputReader(inputLines)

	err := parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})

	assert.NotNil(t, err, "expected error")
	assert.Equal(t, "parse error on line 1, column 65: extraneous or missing \" in quoted-field", err.Error())
}

func TestSendJsonLinesRequests(t *testing.T) {
	requestsWithContext := make(chan parser.RequestWithContext, 3)
	defer close(requestsWithContext)

	inputLines := `
        { "url": "https://ex.com/bar", "context": ["foo", "quoted content"] }
		{ "url": "https://ex.com/qux", "method": "POST", "context": { "quux": "  \"quoted with whitespace\"  ", "corge": 456 } }
		{ "url": "https://ex.com/123", "method": "DELETE", "context": "baz" }
    `

	var in = trimmedInputReader(inputLines)

	err := parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})
	assert.Nil(t, err, "expected no error")

	expectedResults := []struct {
		url     string
		context interface{}
		method  string
	}{
		{"https://ex.com/bar", []interface{}{"foo", "quoted content"}, "GET"},
		{"https://ex.com/qux", map[string]interface{}{"quux": "  \"quoted with whitespace\"  ", "corge": float64(456)}, "POST"},
		{"https://ex.com/123", "baz", "DELETE"},
	}

	for _, expectedResult := range expectedResults {
		requestWithContext := <-requestsWithContext
		request := requestWithContext.Request
		requestContext := requestWithContext.RequestContext

		assert.Equal(t, expectedResult.url, request.URL.String(), "expected url")
		assert.Equal(t, expectedResult.context, requestContext, "expected context")
		assert.Equal(t, expectedResult.method, request.Method, "expected method")
	}
}

func TestSendJsonLinesRequestsMissingUrl(t *testing.T) {
	requestsWithContext := make(chan parser.RequestWithContext, 1)
	defer close(requestsWithContext)

	// missing `url` field
	inputLines := ` { "noturl": "https://ex.com/bar", "context": ["foo", "quoted content"] }`

	var in = trimmedInputReader(inputLines)

	err := parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})

	assert.NotNil(t, err, "expected error")
	assert.Equal(t, "missing url property: { \"noturl\": \"https://ex.com/bar\", \"context\": [\"foo\", \"quoted content\"] }", err.Error())
}

func TestSendJsonLinesRequestsMalformedJson(t *testing.T) {
	requestsWithContext := make(chan parser.RequestWithContext, 1)
	defer close(requestsWithContext)

	// missing trailing `}`
	inputLines := ` { "url": "https://ex.com/bar", "context": ["foo", "quoted content"]`

	var in = trimmedInputReader(inputLines)

	err := parser.SendRequests(requestsWithContext, in, "GET", []config.RequestHeader{})

	assert.NotNil(t, err, "expected error")
	assert.Equal(t, "unexpected end of JSON input: { \"url\": \"https://ex.com/bar\", \"context\": [\"foo\", \"quoted content\"]", err.Error())
}

func TestSendJsonLinesAddGivenHeaders(t *testing.T) {
	requestsWithContext := make(chan parser.RequestWithContext, 1)
	defer close(requestsWithContext)

	inputLines := `{ "url": "https://ex.com/123", "method": "DELETE", "headers": { "X-Bar": "corge" }, "context": "baz" }`

	var in = trimmedInputReader(inputLines)

	staticHeaders := []config.RequestHeader{{Key: "X-Static", Value: "foo"}}

	err := parser.SendRequests(requestsWithContext, in, "GET", staticHeaders)

	assert.Nil(t, err, "expected no error")

	requestWithContext := <-requestsWithContext
	request := requestWithContext.Request
	requestContext := requestWithContext.RequestContext

	assert.Equal(t, "https://ex.com/123", request.URL.String(), "expected url")
	assert.Equal(t, "DELETE", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")
	assert.Equal(t, request.Header["X-Static"][0], "foo")
	assert.Equal(t, request.Header["X-Bar"][0], "corge")
	assert.Equal(t, "baz", requestContext, "expected context")
}

func TestSendJsonLinesGivenHeadersOverrideStaticHeaders(t *testing.T) {
	requestsWithContext := make(chan parser.RequestWithContext, 1)
	defer close(requestsWithContext)

	inputLines := `{ "url": "https://ex.com/123", "method": "DELETE", "headers": { "X-Bar": "corge" }, "context": "baz" }`

	var in = trimmedInputReader(inputLines)

	staticHeaders := []config.RequestHeader{{Key: "X-Bar", Value: "foo"}}

	err := parser.SendRequests(requestsWithContext, in, "GET", staticHeaders)

	assert.Nil(t, err, "expected no error")

	requestWithContext := <-requestsWithContext
	request := requestWithContext.Request
	requestContext := requestWithContext.RequestContext

	assert.Equal(t, "https://ex.com/123", request.URL.String(), "expected url")
	assert.Equal(t, "DELETE", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")
	assert.Equal(t, request.Header["X-Bar"][0], "corge")
	assert.Equal(t, "baz", requestContext, "expected context")
}

// TODO allow "body" to be specified in the JSON line
// body defaults to `raw` (is this a string?), could also be JSON, `escaped`, or `base64`

// TODO then get the request context passing through to the response output

func trimmedInputReader(s string) io.Reader {
	lines := strings.Split(s, "\n")
	var trimmedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 {
			trimmedLines = append(trimmedLines, trimmedLine)
		}
	}
	return strings.NewReader(strings.Join(trimmedLines, "\n"))
}
