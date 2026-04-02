package execcontext

import (
	"io"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
)

func TestNewDefaults(t *testing.T) {
	conf := config.New()
	ctx, err := New(conf, strings.NewReader(""), io.Discard, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, 1*time.Second, ctx.BaseRetryDelayDuration)
	assert.Equal(t, 10*time.Second, ctx.ConnectTimeoutDuration)
	assert.Equal(t, 1, ctx.RequestWorkers)
	assert.Equal(t, 1, ctx.ResponseWorkers)
	assert.Equal(t, 0, ctx.Retries)
	assert.Equal(t, math.MaxInt32, ctx.ThrottlePerSecond)
	assert.Equal(t, false, ctx.WriteFiles)
	assert.Equal(t, false, ctx.Insecure)
	assert.Equal(t, false, ctx.JsonEnvelope)
	assert.NotNil(t, ctx.Logger)
}

func TestNewWithThrottle(t *testing.T) {
	conf := config.New()
	conf.ThrottlePerSecond = 100
	ctx, err := New(conf, strings.NewReader(""), io.Discard, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, 100, ctx.ThrottlePerSecond)
}

func TestNewNegativeThrottleUsesDefault(t *testing.T) {
	conf := config.New()
	conf.ThrottlePerSecond = -1
	ctx, err := New(conf, strings.NewReader(""), io.Discard, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, math.MaxInt32, ctx.ThrottlePerSecond)
}

func TestNewZeroWorkersDefaultsToOne(t *testing.T) {
	conf := config.New()
	conf.RequestWorkers = 0
	ctx, err := New(conf, strings.NewReader(""), io.Discard, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, 1, ctx.RequestWorkers)
}

func TestNewNegativeWorkersDefaultsToOne(t *testing.T) {
	conf := config.New()
	conf.RequestWorkers = -5
	ctx, err := New(conf, strings.NewReader(""), io.Discard, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, 1, ctx.RequestWorkers)
}

func TestNewWithBaseDirectory(t *testing.T) {
	conf := config.New()
	conf.BaseDirectory = "/tmp/output"
	ctx, err := New(conf, strings.NewReader(""), io.Discard, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, true, ctx.WriteFiles)
	assert.Equal(t, "/tmp/output", ctx.BaseDirectory)
}

func TestNewWithRequestFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "ganda-test-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("https://example.com\n")
	tmpFile.Close()

	conf := config.New()
	conf.RequestFilename = tmpFile.Name()
	ctx, err := New(conf, strings.NewReader("stdin"), io.Discard, io.Discard)

	assert.NoError(t, err)
	// In should be replaced with the file reader, not stdin
	assert.NotEqual(t, "stdin", ctx.In)
}

func TestNewWithMissingRequestFile(t *testing.T) {
	conf := config.New()
	conf.RequestFilename = "/nonexistent/file.txt"
	_, err := New(conf, strings.NewReader(""), io.Discard, io.Discard)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unable to open specified file")
}

func TestNewPassesIOHandles(t *testing.T) {
	in := strings.NewReader("test input")
	conf := config.New()
	ctx, err := New(conf, in, io.Discard, io.Discard)

	assert.NoError(t, err)
	assert.Equal(t, in, ctx.In)
}
