package cli

import (
	"fmt"
	"github.com/tednaleid/ganda/config"
	"net/http"
	"testing"
)

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

func TestSilentOutput(t *testing.T) {
	t.Parallel()
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello ", r.URL.Path)
	}))
	defer server.Close()

	runResults, _ := RunApp([]string{"ganda", "-s"}, server.stubStdinUrl("foo/1"))

	runResults.assert(
		t,
		"Hello /foo/1\n",
		"",
	)
}

func TestResponseBody(t *testing.T) {
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello ", r.URL.Path)
	}))
	defer server.Server.Close()

	testCases := []struct {
		name         string
		responseBody config.ResponseBodyType
		expected     string
	}{
		{"raw", config.Raw, "Hello /bar\n"},
		{"discard", config.Discard, ""},
		{"escaped", config.Escaped, "\"Hello /bar\"\n"},
		{"base64", config.Base64, "SGVsbG8gL2Jhcg==\n"},
		{"sha256", config.Sha256, "13a05f3ce0f3edc94bdeee3783c969dfb27c234b6dd98ce7fd004ffc69a45ece\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runResults, _ := RunApp([]string{"ganda", "-B", tc.name}, server.stubStdinUrl("bar"))
			url := server.urlFor("bar")

			runResults.assert(t, tc.expected, "Response: 200 "+url+"\n")
		})
	}
}

func TestResponseBodyWithJsonEnvelope(t *testing.T) {
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "{ \"foo\": \"", r.URL.Path+"\" }")
	}))
	defer server.Server.Close()

	testCases := []struct {
		name         string
		responseBody config.ResponseBodyType
		expected     string
	}{
		{"raw", config.Raw, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 200, \"body\": { \"foo\": \"/bar\" } }\n"},
		{"discard", config.Discard, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 200, \"body\": null }\n"},
		{"escaped", config.Escaped, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 200, \"body\": \"\"{ \\\"foo\\\": \\\"/bar\\\" }\"\" }\n"},
		{"base64", config.Base64, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 200, \"body\": \"eyAiZm9vIjogIi9iYXIiIH0=\" }\n"},
		{"sha256", config.Sha256, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 200, \"body\": \"f660cd1420c6acd9408932b9983909c26ab6cb21ffb40525670a7b7aa67092ec\" }\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runResults, _ := RunApp([]string{"ganda", "-J", "-B", tc.name}, server.stubStdinUrl("bar"))
			url := server.urlFor("bar")

			runResults.assert(t, tc.expected, "Response: 200 "+url+"\n")
		})
	}
}

func TestErrorWithJsonEnvelope(t *testing.T) {
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Server.Close()

	testCases := []struct {
		name         string
		responseBody config.ResponseBodyType
		expected     string
	}{
		{"raw", config.Raw, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 404, \"body\": null }\n"},
		{"discard", config.Discard, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 404, \"body\": null }\n"},
		{"escaped", config.Escaped, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 404, \"body\": null }\n"},
		{"base64", config.Base64, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 404, \"body\": null }\n"},
		{"sha256", config.Sha256, "{ \"url\": \"" + server.urlFor("bar") + "\", \"code\": 404, \"body\": null }\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runResults, _ := RunApp([]string{"ganda", "-J", "-B", tc.name}, server.stubStdinUrl("bar"))
			url := server.urlFor("bar")

			runResults.assert(t, tc.expected, "Response: 404 "+url+"\n")
		})
	}
}

func TestJsonLinesContextWithJsonEnvelope(t *testing.T) {
	server := NewHttpServerStub(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "")
	}))
	defer server.Server.Close()

	url := server.urlFor("bar")

	inputLines := `
        { "url": "` + url + `", "context": ["foo", "quoted content"] }
		{ "url": "` + url + `", "method": "POST", "context": { "quux": "  \"quoted with whitespace\"  ", "corge": 456 } }
		{ "url": "` + url + `", "method": "DELETE", "context": "baz" }
    `

	runResults, _ := RunApp([]string{"ganda", "-J"}, trimmedInputReader(inputLines))

	expectedOutput := trimIndentKeepTrailingNewline(`
		{ "url": "` + url + `", "code": 200, "body": null, "context": ["foo","quoted content"] }
		{ "url": "` + url + `", "code": 200, "body": null, "context": {"corge":456,"quux":"  \"quoted with whitespace\"  "} }
		{ "url": "` + url + `", "code": 200, "body": null, "context": "baz" }
	`)

	expectedLog := trimIndentKeepTrailingNewline(`
		Response: 200 ` + url + `
		Response: 200 ` + url + `
		Response: 200 ` + url + `
	`)

	runResults.assert(t, expectedOutput, expectedLog)
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
		"{ \"url\": \""+server.urlFor("bar")+"\", \"code\": 404, \"body\": null }\n",
		"Response: 404 "+server.urlFor("bar")+"\n",
	)
}

// TODO test the file saving version of this
