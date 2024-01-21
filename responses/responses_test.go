package responses

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"net/http"
	"testing"
)

func TestRawOutput(t *testing.T) {
	context := &execcontext.Context{
		ResponseBody: config.Raw,
	}

	responseFn := determineEmitResponseWithContextFn(context)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponse("hello world")
	writeCloser := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, writeCloser)

	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "hello world", writeCloser.ToString())
}

func TestDiscardOutput(t *testing.T) {
	context := &execcontext.Context{
		ResponseBody: config.Discard,
	}

	responseFn := determineEmitResponseWithContextFn(context)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponse("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "", out.ToString())
}

func TestBase64Output(t *testing.T) {
	context := &execcontext.Context{
		ResponseBody: config.Base64,
	}

	responseFn := determineEmitResponseWithContextFn(context)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponse("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "aGVsbG8gd29ybGQ=", out.ToString())
}

func TestSha256Output(t *testing.T) {
	context := &execcontext.Context{
		ResponseBody: config.Sha256,
	}

	responseFn := determineEmitResponseWithContextFn(context)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponse("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())

	// if testing with "echo" be sure to use the -n flag to not include the newline
	// echo -n "hello world" | shasum -a 256
	// b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9  -
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", out.ToString())

	// ensure that when called a second time, we get the same answer and that the hasher can be reused
	mockResponse2 := NewMockResponse("hello world")
	out2 := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse2.Response}, out2)
	assert.True(t, mockResponse2.BodyClosed())
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", out2.ToString())
}

type MockResponse struct {
	*http.Response
	mockBody *MockReadCloser
}

func (mr *MockResponse) BodyClosed() bool {
	return mr.mockBody.Closed
}

func NewMockResponse(body string) *MockResponse {
	mockReadCloser := &MockReadCloser{
		Reader: bytes.NewReader([]byte(body)),
		Closed: false,
	}
	return &MockResponse{
		Response: &http.Response{
			Body: mockReadCloser,
		},
		mockBody: mockReadCloser,
	}
}

type MockReadCloser struct {
	*bytes.Reader
	Closed bool
}

func (mrc *MockReadCloser) Close() error {
	mrc.Closed = true
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
