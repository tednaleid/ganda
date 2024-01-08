package cli

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestRequestHappyPathHasDefaultHeaders(t *testing.T) {
	t.Parallel()
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// default headers added by http client
		assert.Equal(t, r.Header["User-Agent"][0], "Go-http-client/1.1", "User-Agent header")
		assert.Equal(t, r.Header["Connection"][0], "keep-alive", "Connection header")
		assert.Equal(t, r.Header["Accept-Encoding"][0], "gzip", "Accept-Encoding header")
		fmt.Fprint(w, "Hello ", r.URL.Path)
	}))
	defer server.Close()

	runResults, _ := RunApp([]string{"ganda"}, server.stubStdinUrl("foo/1"))

	runResults.assert(
		t,
		"Hello /foo/1\n",
		"Response: 200 "+server.urlFor("foo/1")+"\n",
	)
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		fmt.Fprint(w, "Should not get this, should time out first")
	}))
	defer server.Server.Close()

	runResults, _ := RunApp([]string{"ganda", "--connect-timeout-ms", "1"}, server.stubStdinUrl("bar"))

	url := server.urlFor("bar")

	runResults.assert(
		t,
		"",
		url+" Error: Get \""+url+"\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)\n",
	)
}

func TestRetryEnabledShouldRetry5XX(t *testing.T) {
	t.Parallel()
	requests := 0
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(500)
		} else {
			fmt.Fprint(w, "Retried request")
		}
	}))
	defer server.Server.Close()

	runResults, _ := RunApp([]string{"ganda", "--retry", "1", "--base-retry-ms", "1"}, server.stubStdinUrl("bar"))

	url := server.urlFor("bar")

	assert.Equal(t, 2, requests, "expected a failed request followed by a successful one")
	runResults.assert(
		t,
		"Retried request\n",
		"Response: 500 "+url+"\nResponse: 200 "+url+"\n",
	)
}

func TestRunningOutOfRetriesShouldStopProcessing(t *testing.T) {
	t.Parallel()
	requests := 0
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(500)
	}))
	defer server.Server.Close()

	runResults, _ := RunApp([]string{"ganda", "--retry", "2", "--base-retry-ms", "1"}, server.stubStdinUrl("bar"))

	url := server.urlFor("bar")

	assert.Equal(t, 3, requests, "3 total requests (original and 2 retries), all failed so expecting error")
	runResults.assert(
		t,
		"",
		"Response: 500 "+url+"\nResponse: 500 "+url+"\nResponse: 500 "+url+"\n",
	)
}

func TestRetryEnabledShouldNotRetry4XX(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(400)
	}))
	defer server.Server.Close()

	runResults, _ := RunApp([]string{"ganda", "--retry", "1", "--base-retry-ms", "1"}, server.stubStdinUrl("bar"))

	url := server.urlFor("bar")

	assert.Equal(t, 1, requestCount, "we shouldn't retry 4xx errors")
	runResults.assert(t,
		"",
		"Response: 400 "+url+"\n")
}

func TestRetryEnabledShouldRetryTimeout(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// for the first request, we sleep longer than it takes to timeout
			time.Sleep(20 * time.Millisecond)
		}
		fmt.Fprint(w, "Request ", requestCount)
	}))
	defer server.Server.Close()

	runResults, _ := RunApp([]string{"ganda", "--connect-timeout-ms", "10", "--retry", "1", "--base-retry-ms", "1"}, server.stubStdinUrl("bar"))
	url := server.urlFor("bar")

	//assert.Equal(t, 2, requestCount, "expected a second request")
	runResults.assert(t,
		"Request 2\n",
		url+" Error: Get \""+url+"\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)\nResponse: 200 "+url+"\n")
}

func TestAddHeadersToRequestCreatesCanonicalKeys(t *testing.T) {
	t.Parallel()
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// turns to uppercase versions for header key when transmitted
		assert.Equal(t, r.Header["Foo"][0], "bar", "foo header")
		assert.Equal(t, r.Header["X-Baz"][0], "qux", "baz header")
		fmt.Fprint(w, "Hello ", r.URL.Path)
	}))
	defer server.Server.Close()

	runResults, _ := RunApp([]string{"ganda", "-H", "foo: bar", "-H", "x-baz: qux"}, server.stubStdinUrl("bar"))
	url := server.urlFor("bar")

	runResults.assert(t,
		"Hello /bar\n",
		"Response: 200 "+url+"\n")
}
