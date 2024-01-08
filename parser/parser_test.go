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
	firstUrl := "https://example.com/bar"
	secondUrl := "https://example.com/qux"

	var in = inputReader([]string{firstUrl, secondUrl})
	var requestMethod = "GET"
	var requestHeaders []config.RequestHeader

	go parser.SendRequests(requestsWithContext, in, requestMethod, requestHeaders)

	requestWithContext := <-requestsWithContext
	request := requestWithContext.Request
	requestContext := requestWithContext.RequestContext

	assert.Equal(t, firstUrl, request.URL.String(), "expected url")
	assert.Equal(t, "GET", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")
	assert.Equal(t, []string{}, requestContext, "expected nil context")

	secondRequestWithContext := <-requestsWithContext
	secondRequest := secondRequestWithContext.Request
	secondRequestContext := secondRequestWithContext.RequestContext

	assert.Equal(t, secondUrl, secondRequest.URL.String(), "expected url")
	assert.Equal(t, []string{}, secondRequestContext, "expected nil context")

	close(requestsWithContext)
}

// TODO start here: test sending context through

func inputReader(urls []string) io.Reader {
	stringUrls := strings.Join(urls, "\n")
	return strings.NewReader(stringUrls)
}
