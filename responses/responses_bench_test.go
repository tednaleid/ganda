package responses

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/tednaleid/ganda/config"
)

func mockResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request: &http.Request{
			Method: "GET",
			Host:   "localhost",
		},
	}
}

func mockResponseWithContext(body string) *ResponseWithContext {
	resp := mockResponse(body) //nolint:bodyclose // caller closes via emit function
	resp.Request, _ = http.NewRequest("GET", "http://localhost:8080/test", nil)
	return &ResponseWithContext{
		Response:       resp,
		RequestContext: "test-context",
	}
}

func BenchmarkEmitRawBody(b *testing.B) {
	body := strings.Repeat("hello world ", 100)
	out := io.Discard

	for b.Loop() {
		resp := mockResponse(body) //nolint:bodyclose // closed inside emitRawBody
		emitRawBody(resp, out)
	}
}

func BenchmarkEmitBase64Body(b *testing.B) {
	body := strings.Repeat("hello world ", 100)
	out := io.Discard

	for b.Loop() {
		resp := mockResponse(body) //nolint:bodyclose // closed inside emitBase64Body
		emitBase64Body(resp, out)
	}
}

func BenchmarkEmitSha256Body(b *testing.B) {
	body := strings.Repeat("hello world ", 100)
	out := io.Discard
	fn := emitSha256BodyFn() //nolint:bodyclose // body closed inside returned closure

	for b.Loop() {
		resp := mockResponse(body) //nolint:bodyclose // closed inside fn
		fn(resp, out)
	}
}

func BenchmarkEmitEscapedBody(b *testing.B) {
	body := `{"key": "value", "nested": {"arr": [1, 2, 3]}}`
	out := io.Discard
	fn := emitEscapedBodyFn() //nolint:bodyclose // body closed inside returned closure

	for b.Loop() {
		resp := mockResponse(body) //nolint:bodyclose // closed inside fn
		fn(resp, out)
	}
}

func BenchmarkEmitJsonEnvelope(b *testing.B) {
	body := `{"key": "value"}`
	out := new(bytes.Buffer)
	fn := determineEmitJsonResponseWithContextFn(config.Raw)

	for b.Loop() {
		out.Reset()
		rwc := mockResponseWithContext(body) //nolint:bodyclose // closed inside emit function
		fn(rwc, out)
	}
}

func BenchmarkDirectoryForFile(b *testing.B) {
	for b.Loop() {
		directoryForFile("/tmp/output", "http-localhost-8080-api-v1-users-12345", 2)
	}
}
