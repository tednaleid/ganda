package config

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaults(t *testing.T) {
	conf := New()

	assert.Equal(t, 1000, conf.BaseRetryDelayMillis)
	assert.Equal(t, 10_000, conf.ConnectTimeoutMillis)
	assert.Equal(t, false, conf.Color)
	assert.Equal(t, false, conf.Insecure)
	assert.Equal(t, false, conf.JsonEnvelope)
	assert.Equal(t, "GET", conf.RequestMethod)
	assert.Equal(t, 1, conf.RequestWorkers)
	assert.Equal(t, Raw, conf.ResponseBody)
	assert.Equal(t, 0, conf.Retries)
	assert.Equal(t, false, conf.Silent)
	assert.Equal(t, 0, conf.SubdirLength)
	assert.Equal(t, math.MaxInt32, conf.ThrottlePerSecond)
}

func TestNewRequestHeaderValid(t *testing.T) {
	header, err := NewRequestHeader("Content-Type: application/json")
	assert.NoError(t, err)
	assert.Equal(t, "Content-Type", header.Key)
	assert.Equal(t, "application/json", header.Value)
}

func TestNewRequestHeaderTrimsWhitespace(t *testing.T) {
	header, err := NewRequestHeader("  X-Custom  :  some value  ")
	assert.NoError(t, err)
	assert.Equal(t, "X-Custom", header.Key)
	assert.Equal(t, "some value", header.Value)
}

func TestNewRequestHeaderMissingColon(t *testing.T) {
	_, err := NewRequestHeader("BadHeader")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing ':'")
}

func TestNewRequestHeaderValueWithColons(t *testing.T) {
	header, err := NewRequestHeader("Authorization: Bearer token:with:colons")
	assert.NoError(t, err)
	assert.Equal(t, "Authorization", header.Key)
	assert.Equal(t, "Bearer token:with:colons", header.Value)
}

func TestConvertRequestHeaders(t *testing.T) {
	headers, err := ConvertRequestHeaders([]string{
		"Content-Type: application/json",
		"Authorization: Bearer abc123",
	})
	assert.NoError(t, err)
	assert.Len(t, headers, 2)
	assert.Equal(t, "Content-Type", headers[0].Key)
	assert.Equal(t, "Authorization", headers[1].Key)
}

func TestConvertRequestHeadersEmpty(t *testing.T) {
	headers, err := ConvertRequestHeaders([]string{})
	assert.NoError(t, err)
	assert.Len(t, headers, 0)
}

func TestConvertRequestHeadersReturnsErrorOnInvalid(t *testing.T) {
	_, err := ConvertRequestHeaders([]string{
		"Good: header",
		"BadHeader",
	})
	assert.Error(t, err)
}

func TestResponseBodyTypeValues(t *testing.T) {
	assert.Equal(t, ResponseBodyType("base64"), Base64)
	assert.Equal(t, ResponseBodyType("discard"), Discard)
	assert.Equal(t, ResponseBodyType("escaped"), Escaped)
	assert.Equal(t, ResponseBodyType("sha256"), Sha256)
	assert.Equal(t, ResponseBodyType("raw"), Raw)
}
