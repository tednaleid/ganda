package responses

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"net/http"
	"net/url"
	"testing"
)

func TestRawOutput(t *testing.T) {
	responseFn := determineEmitResponseFn(config.Raw)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	writeCloser := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, writeCloser)

	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "hello world", writeCloser.ToString())
}

func TestRawOutputJSON(t *testing.T) {
	responseFn := determineEmitJsonResponseWithContextFn(config.Raw)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("\"hello world\"")
	writeCloser := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response, RequestContext: nil}, writeCloser)

	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"hello world\" }", writeCloser.ToString())
}

func TestEscapedOutput(t *testing.T) {
	responseFn := determineEmitResponseFn(config.Escaped)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	writeCloser := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, writeCloser)

	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "\"hello world\"", writeCloser.ToString())
}

func TestEscapedOutputJSON(t *testing.T) {
	responseFn := determineEmitJsonResponseWithContextFn(config.Escaped)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("\"hello world\"")
	writeCloser := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response, RequestContext: nil}, writeCloser)

	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"\\\"hello world\\\"\" }", writeCloser.ToString())
}

func TestDiscardOutput(t *testing.T) {
	responseFn := determineEmitResponseFn(config.Discard)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "", out.ToString())
}

func TestDiscardOutputJSON(t *testing.T) {
	responseFn := determineEmitJsonResponseWithContextFn(config.Discard)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": null }", out.ToString())
}

func TestBase64Output(t *testing.T) {
	responseFn := determineEmitResponseFn(config.Base64)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "aGVsbG8gd29ybGQ=", out.ToString())
}

func TestBase64OutputJSON(t *testing.T) {
	responseFn := determineEmitJsonResponseWithContextFn(config.Base64)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())
	assert.Equal(t, "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"aGVsbG8gd29ybGQ=\" }", out.ToString())
}

func TestSha256Output(t *testing.T) {
	responseFn := determineEmitResponseFn(config.Sha256)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())

	// if testing with "echo" be sure to use the -n flag to not include the newline
	// echo -n "hello world" | shasum -a 256
	// b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9  -
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", out.ToString())

	// ensure that when called a second time, we get the same answer and that the hasher can be reused
	mockResponse2 := NewMockResponseBodyOnly("hello world")
	out2 := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse2.Response}, out2)
	assert.True(t, mockResponse2.BodyClosed())
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", out2.ToString())
}

func TestSha256OutputJSON(t *testing.T) {
	responseFn := determineEmitJsonResponseWithContextFn(config.Sha256)
	assert.NotNil(t, responseFn)

	mockResponse := NewMockResponseBodyOnly("hello world")
	out := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse.Response}, out)
	assert.True(t, mockResponse.BodyClosed())

	// if testing with "echo" be sure to use the -n flag to not include the newline
	// echo -n "hello world" | shasum -a 256
	// b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9  -
	assert.Equal(t,
		"{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\" }",
		out.ToString())

	// ensure that when called a second time, we get the same answer and that the hasher can be reused
	mockResponse2 := NewMockResponseBodyOnly("hello world")
	out2 := NewMockWriteCloser()

	responseFn(&ResponseWithContext{Response: mockResponse2.Response}, out2)
	assert.True(t, mockResponse2.BodyClosed())
	assert.Equal(t,
		"{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\" }",
		out2.ToString())
}

func TestRawOutputWithRequestContextJSON(t *testing.T) {
	responseFn := determineEmitJsonResponseWithContextFn(config.Raw)
	assert.NotNil(t, responseFn)

	testCases := []struct {
		name           string
		requestContext interface{}
		expectedOutput string
	}{
		{
			name:           "string RequestContext",
			requestContext: "a context string",
			expectedOutput: "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"hello world\", \"context\": \"a context string\" }",
		},
		{
			name:           "list of strings RequestContext",
			requestContext: []string{"context1", "context2"},
			expectedOutput: "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"hello world\", \"context\": [\"context1\",\"context2\"] }",
		},
		{
			name:           "map RequestContext",
			requestContext: map[string]string{"key1": "value1", "key2": "value2"},
			expectedOutput: "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"hello world\", \"context\": {\"key1\":\"value1\",\"key2\":\"value2\"} }",
		},
		{
			name: "object RequestContext",
			requestContext: struct {
				Field1 string `json:"field1"`
				Field2 int    `json:"field2"`
			}{Field1: "value1", Field2: 2},
			expectedOutput: "{ \"url\": \"http://example.com\", \"code\": 200, \"body\": \"hello world\", \"context\": {\"field1\":\"value1\",\"field2\":2} }",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockResponse := NewMockResponseBodyOnly("\"hello world\"")
			writeCloser := NewMockWriteCloser()

			responseFn(&ResponseWithContext{Response: mockResponse.Response, RequestContext: tc.requestContext}, writeCloser)

			assert.True(t, mockResponse.BodyClosed())
			assert.Equal(t, tc.expectedOutput, writeCloser.ToString())
		})
	}
}

type MockResponse struct {
	*http.Response
	mockBody *MockReadCloser
}

func (mr *MockResponse) BodyClosed() bool {
	return mr.mockBody.Closed
}

func NewMockResponseBodyOnly(body string) *MockResponse {
	return NewMockResponse(body, "http://example.com", 200)
}

func NewMockResponse(body string, fullUrl string, statusCode int) *MockResponse {
	parsedURL, _ := url.Parse(fullUrl)

	mockReadCloser := &MockReadCloser{
		Reader: bytes.NewReader([]byte(body)),
		Closed: false,
	}
	return &MockResponse{
		Response: &http.Response{
			Body:       mockReadCloser,
			StatusCode: statusCode,
			Request: &http.Request{
				URL: parsedURL,
			},
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
