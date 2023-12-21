package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/logger"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type HttpServerStub struct {
	Server            *httptest.Server
	BaseURL           string
	StandardOutBuffer *bytes.Buffer
	StandardOutStub   io.Writer
	LogBuffer         *bytes.Buffer
	LoggerStub        *log.Logger
}

// The passed in handler function can verify the request and write a response given that input
func NewHttpServerStub(handler http.Handler) *HttpServerStub {
	server := httptest.NewServer(handler)

	httpServerStub := HttpServerStub{
		Server:            server,
		BaseURL:           server.URL,
		StandardOutBuffer: new(bytes.Buffer),
		LogBuffer:         new(bytes.Buffer),
	}

	httpServerStub.StandardOutStub = httpServerStub.StandardOutBuffer
	httpServerStub.LoggerStub = log.New(httpServerStub.LogBuffer, "", 0)

	return &httpServerStub
}

func TestSimpleRequest(t *testing.T) {
	t.Parallel()
	RunApp([]string{"ganda", "-h"})
}

func TestRequestHappyPathHeadersAndResults(t *testing.T) {
	t.Parallel()
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// default headers added by http client
		assert.Equal(t, r.Header["User-Agent"][0], "Go-http-client/1.1", "User-Agent header")
		assert.Equal(t, r.Header["Connection"][0], "keep-alive", "Connection header")
		assert.Equal(t, r.Header["Accept-Encoding"][0], "gzip", "Accept-Encoding header")
		fmt.Fprint(w, "Hello ", r.URL.Path)
	}))
	defer httpServerStub.Server.Close()

	context := newTestContext(httpServerStub, []string{httpServerStub.BaseURL + "/bar"})

	processRequests(context)

	assertOutput(t, httpServerStub,
		"Hello /bar\n",
		"Response: 200 "+httpServerStub.BaseURL+"/bar\n")
}

func TestResponseHasJsonEnvelopeWhenRequested(t *testing.T) {
	t.Parallel()
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{ \"foo\": true }")
	}))
	defer httpServerStub.Server.Close()

	context := newTestContext(httpServerStub, []string{httpServerStub.BaseURL + "/bar"})
	context.JsonEnvelope = true

	processRequests(context)

	assertOutput(t, httpServerStub,
		"{ \"url\": \""+httpServerStub.BaseURL+"/bar\", \"code\": 200, \"length\": 15, \"body\": { \"foo\": true } }\n",
		"Response: 200 "+httpServerStub.BaseURL+"/bar\n")
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer httpServerStub.Server.Close()

	context := newTestContext(httpServerStub, []string{httpServerStub.BaseURL + "/bar"})

	processRequests(context)

	assertOutput(t, httpServerStub,
		"",
		"Response: 404 "+httpServerStub.BaseURL+"/bar\n")
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		fmt.Fprint(w, "Should not get this, should time out first")
	}))
	defer httpServerStub.Server.Close()

	url := httpServerStub.BaseURL + "/bar"
	c := newTestContext(httpServerStub, []string{url})
	c.ConnectTimeoutDuration = time.Duration(1) * time.Millisecond

	processRequests(c)

	assertOutput(t, httpServerStub,
		"",
		url+" Error: Get \""+url+"\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)\n")
}

func TestRetryEnabledShouldRetry5XX(t *testing.T) {
	t.Parallel()
	requests := 0
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(500)
		} else {
			fmt.Fprint(w, "Retried request")
		}
	}))
	defer httpServerStub.Server.Close()

	url := httpServerStub.BaseURL + "/bar"
	context := newTestContext(httpServerStub, []string{url})
	context.Retries = 1

	processRequests(context)

	assert.Equal(t, 2, requests, "had a failed request followed by a successful one")
	assertOutput(t, httpServerStub,
		"Retried request\n",
		"Response: 500 "+httpServerStub.BaseURL+"/bar (1)\nResponse: 200 "+httpServerStub.BaseURL+"/bar\n")
}

func TestRunningOutOfRetriesShouldShowError(t *testing.T) {
	t.Parallel()
	requests := 0
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(500)
	}))
	defer httpServerStub.Server.Close()

	url := httpServerStub.BaseURL + "/bar"
	context := newTestContext(httpServerStub, []string{url})
	context.Retries = 2

	processRequests(context)

	assert.Equal(t, 3, requests, "3 total requests (original and 2 retries), all failed so expecting error")
	assertOutput(t, httpServerStub,
		"",
		"Response: 500 "+httpServerStub.BaseURL+"/bar (1)\nResponse: 500 "+httpServerStub.BaseURL+"/bar (2)\nResponse: 500 "+httpServerStub.BaseURL+"/bar\n")
}

func TestRetryEnabledShouldNotRetry4XX(t *testing.T) {
	t.Parallel()
	requestCount := 0
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(400)
	}))
	defer httpServerStub.Server.Close()

	url := httpServerStub.BaseURL + "/bar"
	context := newTestContext(httpServerStub, []string{url})
	context.Retries = 1

	processRequests(context)

	assert.Equal(t, 1, requestCount, "had a failed request")
	assertOutput(t, httpServerStub,
		"",
		"Response: 400 "+httpServerStub.BaseURL+"/bar\n")
}

func TestRetryEnabledShouldRetryTimeout(t *testing.T) {
	t.Parallel()
	requestCount := 0
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		requestCount++
		fmt.Fprint(w, "Request ", requestCount)
	}))
	defer httpServerStub.Server.Close()

	url := httpServerStub.BaseURL + "/bar"
	context := newTestContext(httpServerStub, []string{url})
	context.Retries = 1
	context.ConnectTimeoutDuration = time.Duration(1) * time.Millisecond

	processRequests(context)

	assert.Equal(t, 2, requestCount, "expected a second request")
	assertOutput(t, httpServerStub,
		"Request 2\n",
		httpServerStub.BaseURL+"/bar (1) Error: Get \""+httpServerStub.BaseURL+"/bar\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)\nResponse: 200 "+httpServerStub.BaseURL+"/bar\n")
}

func TestAddHeadersToRequestCreatesCanonicalKeys(t *testing.T) {
	t.Parallel()
	httpServerStub := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// turns to uppercase versions for header key when transmitted
		assert.Equal(t, r.Header["Foo"][0], "bar", "foo header")
		assert.Equal(t, r.Header["X-Baz"][0], "qux", "baz header")
		fmt.Fprint(w, "Hello ", r.URL.Path)
	}))
	defer httpServerStub.Server.Close()

	context := newTestContext(httpServerStub, []string{httpServerStub.BaseURL + "/bar"})

	context.RequestHeaders = []config.RequestHeader{
		{Key: "foo", Value: "bar"},
		{Key: "x-baz", Value: "qux"},
	}

	processRequests(context)

	assertOutput(t, httpServerStub,
		"Hello /bar\n",
		"Response: 200 "+httpServerStub.BaseURL+"/bar\n")
}

func newTestContext(httpServerStub *HttpServerStub, expectedURLPaths []string) *execcontext.Context {
	return &execcontext.Context{
		RequestWorkers:    1,
		ResponseWorkers:   1,
		ThrottlePerSecond: math.MaxInt32,
		RequestScanner:    urlsScanner(expectedURLPaths),
		Out:               httpServerStub.StandardOutStub,
		Logger:            logger.NewPlainLeveledLogger(httpServerStub.LoggerStub),
	}
}

func assertOutput(t *testing.T, httpServerStub *HttpServerStub, expectedStandardOut string, expectedLog string) {
	actualOut := httpServerStub.StandardOutBuffer.String()
	assert.Equal(t, expectedStandardOut, actualOut, "expected stdout")
	actualLog := httpServerStub.LogBuffer.String()
	assert.Equal(t, expectedLog, actualLog, "expected logger stderr")
}

func urlsScanner(urls []string) *bufio.Scanner {
	stringUrls := strings.Join(urls, "\n")
	return bufio.NewScanner(strings.NewReader(stringUrls))
}
