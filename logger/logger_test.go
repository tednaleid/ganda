package logger

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSilentLoggerSuppressesAllOutput(t *testing.T) {
	l := NewSilentLogger()

	// silent logger has no underlying logger, so Info/Warn/Success should not panic
	l.Info("should not appear: %s", "info")
	l.Warn("should not appear: %s", "warn")
	l.Success("should not appear: %s", "success")
}

func TestPlainLoggerInfo(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewPlainLeveledLogger(log.New(buf, "", 0))

	l.Info("hello %s", "world")
	assert.Equal(t, "hello world\n", buf.String())
}

func TestPlainLoggerWarn(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewPlainLeveledLogger(log.New(buf, "", 0))

	l.Warn("warning: %d", 42)
	assert.Equal(t, "warning: 42\n", buf.String())
}

func TestPlainLoggerSuccess(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewPlainLeveledLogger(log.New(buf, "", 0))

	l.Success("ok: %s", "done")
	assert.Equal(t, "ok: done\n", buf.String())
}

func TestColorLoggerWarnAddsColor(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewLeveledLogger(log.New(buf, "", 0))

	l.Warn("error: %s", "bad")
	assert.Contains(t, buf.String(), "\033[31m")
	assert.Contains(t, buf.String(), "error: bad")
	assert.Contains(t, buf.String(), "\033[0m")
}

func TestColorLoggerSuccessAddsColor(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewLeveledLogger(log.New(buf, "", 0))

	l.Success("ok: %s", "good")
	assert.Contains(t, buf.String(), "\033[32m")
	assert.Contains(t, buf.String(), "ok: good")
	assert.Contains(t, buf.String(), "\033[0m")
}

func TestColorLoggerInfoNoColor(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewLeveledLogger(log.New(buf, "", 0))

	l.Info("plain: %s", "text")
	assert.Equal(t, "plain: text\n", buf.String())
	assert.NotContains(t, buf.String(), "\033[")
}

func TestLogResponseSuccess(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewPlainLeveledLogger(log.New(buf, "", 0))

	l.LogResponse(200, "https://example.com")
	assert.Equal(t, "Response: 200 https://example.com\n", buf.String())
}

func TestLogResponseError(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewPlainLeveledLogger(log.New(buf, "", 0))

	l.LogResponse(500, "https://example.com")
	assert.Equal(t, "Response: 500 https://example.com\n", buf.String())
}

func TestLogError(t *testing.T) {
	buf := new(bytes.Buffer)
	l := NewPlainLeveledLogger(log.New(buf, "", 0))

	l.LogError(assert.AnError, "https://example.com")
	assert.Contains(t, buf.String(), "https://example.com")
	assert.Contains(t, buf.String(), "Error:")
}
