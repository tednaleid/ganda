package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/base"
	"log"
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
		"Response:  200 "+scaffold.BaseURL+"/bar\n")
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
		"Response:  404 "+scaffold.BaseURL+"/bar\n")
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	scaffold := NewScaffold(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		fmt.Fprintln(w, "Should not get this, should time out first")
	}))
	defer scaffold.Server.Close()

	url := scaffold.BaseURL + "/bar"
	context := newTestContext(scaffold, []string{url})
	context.ConnectTimeoutDuration = time.Duration(1) * time.Millisecond

	run(context)

	assertOutput(t, scaffold,
		"",
		url+" Error: Get "+url+": net/http: request canceled (Client.Timeout exceeded while awaiting headers)\n")
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

	context.RequestHeaders = []base.RequestHeader{
		{Key: "foo", Value: "bar"},
		{Key: "x-baz", Value: "qux"},
	}

	run(context)

	assertOutput(t, scaffold,
		"Hello /bar\n",
		"Response:  200 "+scaffold.BaseURL+"/bar\n")
}

func newTestContext(scaffold *Scaffold, expectedURLPaths []string) *base.Context {
	return &base.Context{
		RequestWorkers: 1,
		UrlScanner:     urlsScanner(expectedURLPaths),
		Out:            scaffold.StandardOutMock,
		Logger:         scaffold.LoggerMock,
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
