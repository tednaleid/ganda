package responses

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRawOutput(t *testing.T) {
	context := &execcontext.Context{
		ResponseBody: config.Raw,
	}

	responseFn := determineEmitResponseFn(context)
	assert.NotNil(t, responseFn)

	response, responseBody := mockResponseBody("hello world")
	writeCloser := NewMockWriteCloser()

	responseFn(response, writeCloser)

	assert.True(t, responseBody.Closed)
	assert.Equal(t, "hello world", writeCloser.ToString())
}

func TestSha256Output(t *testing.T) {
	context := &execcontext.Context{
		ResponseBody: config.Sha256,
	}

	responseFn := determineEmitResponseFn(context)
	assert.NotNil(t, responseFn)

	response, responseBody := mockResponseBody("hello world")
	out := NewMockWriteCloser()

	responseFn(response, out)

	assert.True(t, responseBody.Closed)
	assert.Equal(t, "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447", out.ToString())
}

func mockResponseBody(body string) (*http.Response, *MockReadCloser) {
	responseBody := &MockReadCloser{ReadCloser: io.NopCloser(strings.NewReader(body)), Closed: false}
	return &http.Response{
		StatusCode: 200,
		Body:       responseBody,
	}, responseBody
}

type MockReadCloser struct {
	ReadCloser io.ReadCloser
	Closed     bool
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	return m.ReadCloser.Read(p)
}

func (m *MockReadCloser) Close() error {
	m.Closed = true
	return nil
}

type MockWriteCloser struct {
	Buffer *bytes.Buffer
	Closed bool
}

func (m *MockWriteCloser) Write(p []byte) (n int, err error) {
	return m.Buffer.Write(p)
}

func (m *MockWriteCloser) Close() error {
	m.Closed = true
	return nil
}

func (m *MockWriteCloser) ToString() string {
	return m.Buffer.String()
}

func NewMockWriteCloser() *MockWriteCloser {
	return &MockWriteCloser{
		Buffer: new(bytes.Buffer),
		Closed: false,
	}
}
