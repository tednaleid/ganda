package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/logger"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type Scaffold struct {
	Server            *httptest.Server
	BaseURL           string
	StandardOutBuffer *bytes.Buffer
	LogBuffer         *bytes.Buffer
	StandardOutMock   *log.Logger
	LoggerMock        *log.Logger
}

func NewScaffold(handler http.Handler) *Scaffold {
	server := httptest.NewServer(handler)

	scaffold := Scaffold{
		Server:            server,
		BaseURL:           server.URL,
		StandardOutBuffer: new(bytes.Buffer),
		LogBuffer:         new(bytes.Buffer),
	}

	scaffold.StandardOutMock = log.New(scaffold.StandardOutBuffer, "", 0)
	scaffold.LoggerMock = log.New(scaffold.LogBuffer, "", 0)

	return &scaffold
}

func TestRequestHappyPathHeadersAndResults(t *testing.T) {
	t.Parallel()
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// default headers added by http client
		assert.Equal(t, r.Header["User-Agent"][0], "Go-http-client/1.1", "User-Agent header")
		assert.Equal(t, r.Header["Connection"][0], "keep-alive", "Connection header")
		assert.Equal(t, r.Header["Accept-Encoding"][0], "gzip", "Accept-Encoding header")
		fmt.Fprintln(w, "Hello", r.URL.Path)
	}))
	defer scaffold.Server.Close()

	context := newTestContext(scaffold, []string{scaffold.BaseURL + "/bar"})

	run(context)

	assertOutput(t, scaffold,
		"Hello /bar\n",
		"Response: 200 "+scaffold.BaseURL+"/bar\n")
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer scaffold.Server.Close()

	context := newTestContext(scaffold, []string{scaffold.BaseURL + "/bar"})

	run(context)

	assertOutput(t, scaffold,
		"\n",
		"Response: 404 "+scaffold.BaseURL+"/bar\n")
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		fmt.Fprintln(w, "Should not get this, should time out first")
	}))
	defer scaffold.Server.Close()

	url := scaffold.BaseURL + "/bar"
	c := newTestContext(scaffold, []string{url})
	c.ConnectTimeoutDuration = time.Duration(1) * time.Millisecond

	run(c)

	assertOutput(t, scaffold,
		"",
		url+" Error: Get "+url+": net/http: request canceled (Client.Timeout exceeded while awaiting headers)\n")
}

func TestRetryEnabledShouldRetry5XX(t *testing.T) {
	t.Parallel()
	requests := 0
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(500)
		} else {
			fmt.Fprintln(w, "Retried request")
		}
	}))
	defer scaffold.Server.Close()

	url := scaffold.BaseURL + "/bar"
	context := newTestContext(scaffold, []string{url})
	context.Retries = 1

	run(context)

	assert.Equal(t, 2, requests, "had a failed request followed by a successful one")
	assertOutput(t, scaffold,
		"Retried request\n",
		"Response: 500 "+scaffold.BaseURL+"/bar (1)\nResponse: 200 "+scaffold.BaseURL+"/bar\n")
}

func TestRunningOutOfRetriesShouldShowError(t *testing.T) {
	t.Parallel()
	requests := 0
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(500)
	}))
	defer scaffold.Server.Close()

	url := scaffold.BaseURL + "/bar"
	context := newTestContext(scaffold, []string{url})
	context.Retries = 2

	run(context)

	assert.Equal(t, 3, requests, "3 total requests (original and 2 retries), all failed so expecting error")
	assertOutput(t, scaffold,
		"\n",
		"Response: 500 "+scaffold.BaseURL+"/bar (1)\nResponse: 500 "+scaffold.BaseURL+"/bar (2)\nResponse: 500 "+scaffold.BaseURL+"/bar\n")
}

func TestRetryEnabledShouldNotRetry4XX(t *testing.T) {
	t.Parallel()
	requestCount := 0
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(400)
	}))
	defer scaffold.Server.Close()

	url := scaffold.BaseURL + "/bar"
	context := newTestContext(scaffold, []string{url})
	context.Retries = 1

	run(context)

	assert.Equal(t, 1, requestCount, "had a failed request")
	assertOutput(t, scaffold,
		"\n",
		"Response: 400 "+scaffold.BaseURL+"/bar\n")
}

func TestRetryEnabledShouldRetryTimeout(t *testing.T) {
	t.Parallel()
	requestCount := 0
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestCount == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		requestCount++
		fmt.Fprintln(w, "Request", requestCount)
	}))
	defer scaffold.Server.Close()

	url := scaffold.BaseURL + "/bar"
	context := newTestContext(scaffold, []string{url})
	context.Retries = 1
	context.ConnectTimeoutDuration = time.Duration(1) * time.Millisecond

	run(context)

	assert.Equal(t, 2, requestCount, "expected a second request")
	assertOutput(t, scaffold,
		"Request 2\n",
		scaffold.BaseURL+"/bar (1) Error: Get "+scaffold.BaseURL+"/bar: net/http: request canceled (Client.Timeout exceeded while awaiting headers)\nResponse: 200 "+scaffold.BaseURL+"/bar\n")
}

func TestAddHeadersToRequestCreatesCanonicalKeys(t *testing.T) {
	t.Parallel()
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// turns to uppercase versions for header key when transmitted
		assert.Equal(t, r.Header["Foo"][0], "bar", "foo header")
		assert.Equal(t, r.Header["X-Baz"][0], "qux", "baz header")
		fmt.Fprintln(w, "Hello", r.URL.Path)
	}))
	defer scaffold.Server.Close()

	context := newTestContext(scaffold, []string{scaffold.BaseURL + "/bar"})

	context.RequestHeaders = []config.RequestHeader{
		{Key: "foo", Value: "bar"},
		{Key: "x-baz", Value: "qux"},
	}

	run(context)

	assertOutput(t, scaffold,
		"Hello /bar\n",
		"Response: 200 "+scaffold.BaseURL+"/bar\n")
}

func newTestContext(scaffold *Scaffold, expectedURLPaths []string) *execcontext.Context {
	return &execcontext.Context{
		RequestWorkers:    1,
		ResponseWorkers:   1,
		ThrottlePerSecond: math.MaxInt32,
		UrlScanner:        urlsScanner(expectedURLPaths),
		Out:               scaffold.StandardOutMock,
		Logger:            logger.NewPlainLeveledLogger(scaffold.LoggerMock),
	}
}

func assertOutput(t *testing.T, scaffold *Scaffold, expectedStandardOut string, expectedLog string) {
	assert.Equal(t, expectedStandardOut, scaffold.StandardOutBuffer.String(), "expected stdout")
	assert.Equal(t, expectedLog, scaffold.LogBuffer.String(), "expected logger stderr")
}

func urlsScanner(urls []string) *bufio.Scanner {
	stringUrls := strings.Join(urls, "\n")
	return bufio.NewScanner(strings.NewReader(stringUrls))
}
