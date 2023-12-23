package parser_test

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/parser"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
)

type parseExpectation struct {
	input        string
	dataTemplate string
	url          string
	body         string
}

var parseStaticBodyList = []parseExpectation{
	{"", "", "", ""},
	{"http://example.com", "", "http://example.com", ""},
	{"http://example.com 123 456", "", "http://example.com", "123 456"},
	{"http://example.com {\"foo\": 123}", "", "http://example.com", "{\"foo\": 123}"},
	{"http://example.com", "%s", "http://example.com", "%s"},
	{"http://example.com 123 456", "%s %s", "http://example.com", "123 456"},
	{"http://example.com 123", "%s %%s", "http://example.com", "123 %s"},
}

func TestParseInputToUrlAndBody(t *testing.T) {
	var url string
	var bodyReader io.Reader

	for _, expectation := range parseStaticBodyList {
		if expectation.dataTemplate == "" {
			url, bodyReader = parser.ParseUrlAndOptionalBody(expectation.input)
		} else {
			url, bodyReader = parser.ParseTemplatedInput(expectation.input, expectation.dataTemplate)
		}

		assert.Equal(t, expectation.url, url)

		if expectation.body == "" {
			assert.Nil(t, bodyReader)
		} else {
			assert.Equal(t, expectation.body, readerToString(bodyReader))
		}
	}
}

func TestSendGetRequestsJustUrls(t *testing.T) {
	t.Parallel()

	requestsChannel := make(chan *http.Request)
	firstUrl := "https://example.com/bar"
	secondUrl := "https://example.com/qux"

	context := &execcontext.Context{
		ThrottlePerSecond: math.MaxInt32,
		In:                inputReader([]string{firstUrl, secondUrl}),
		RequestMethod:     "GET",
	}

	go parser.SendRequests(context, requestsChannel)

	request := <-requestsChannel

	assert.Equal(t, firstUrl, request.URL.String(), "expected url")
	assert.Equal(t, "GET", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")

	secondRequest := <-requestsChannel

	assert.Equal(t, secondUrl, secondRequest.URL.String(), "expected url")

	close(requestsChannel)
}

func TestSendGetRequestsWithBody(t *testing.T) {
	t.Parallel()

	requestsChannel := make(chan *http.Request)
	firstLine := "https://example.com/bar 123"
	secondLine := "https://example.com/qux 456"

	context := &execcontext.Context{
		ThrottlePerSecond: math.MaxInt32,
		In:                inputReader([]string{firstLine, secondLine}),
		RequestMethod:     "POST",
		DataTemplate:      "value: %s",
	}

	go parser.SendRequests(context, requestsChannel)

	request := <-requestsChannel

	assert.Equal(t, "https://example.com/bar", request.URL.String(), "expected url")
	assert.Equal(t, "POST", request.Method, "expected method")
	assert.Equal(t, request.Header["Connection"][0], "keep-alive", "Connection header")
	assert.Equal(t, "value: 123", readerToString(request.Body))

	secondRequest := <-requestsChannel

	assert.Equal(t, "https://example.com/qux", secondRequest.URL.String(), "expected url")
	assert.Equal(t, "value: 456", readerToString(secondRequest.Body))

	close(requestsChannel)
}

func readerToString(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String()
}

func inputReader(urls []string) io.Reader {
	stringUrls := strings.Join(urls, "\n")
	return strings.NewReader(stringUrls)
}
