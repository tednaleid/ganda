package cli

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestRequestColorOutput(t *testing.T) {
	t.Parallel()
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello ", r.URL.Path)
	}))
	defer server.Close()

	runResults, _ := RunApp([]string{"ganda", "--color"}, server.stubStdinUrl("foo/1"))

	runResults.assert(
		t,
		"Hello /foo/1\n",
		"\x1b[32mResponse: 200 "+server.urlFor("foo/1")+"\x1b[0m\n",
	)
}

func TestResponseHasJsonEnvelopeWhenRequested(t *testing.T) {
	t.Parallel()
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{ \"foo\": true }")
	}))
	defer server.Close()

	runResults, _ := RunApp([]string{"ganda", "-J"}, server.stubStdinUrl("bar"))

	runResults.assert(
		t,
		"{ \"url\": \""+server.urlFor("bar")+"\", \"code\": 200, \"length\": 15, \"body\": { \"foo\": true } }\n",
		"Response: 200 "+server.urlFor("bar")+"\n",
	)
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	runResults, _ := RunApp([]string{"ganda", "-J"}, server.stubStdinUrl("bar"))

	runResults.assert(
		t,
		"{ \"url\": \""+server.urlFor("bar")+"\", \"code\": 404, \"length\": 0, \"body\": null }\n",
		"Response: 404 "+server.urlFor("bar")+"\n",
	)
}

// TODO start here convert the rest of the tests in cli_old_test.go to use the new RunApp function
