package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tednaleid/ganda/config"
)

func BenchmarkSendUrlsRequests(b *testing.B) {
	headers := []config.RequestHeader{
		{Key: "Authorization", Value: "Bearer token123"},
	}

	for b.Loop() {
		urls := make([]string, 1000)
		for i := range urls {
			urls[i] = fmt.Sprintf("http://localhost:8080/path/%d", i)
		}
		input := strings.NewReader(strings.Join(urls, "\n"))

		ch := make(chan RequestWithContext, 100)
		go func() {
			for range ch {
			}
		}()

		SendRequests(ch, input, "GET", headers)
		close(ch)
	}
}

func BenchmarkSendJsonLinesRequests(b *testing.B) {
	headers := []config.RequestHeader{}

	for b.Loop() {
		lines := make([]string, 1000)
		for i := range lines {
			lines[i] = fmt.Sprintf(`{"url": "http://localhost:8080/path/%d", "method": "POST", "body": {"key": "value"}}`, i)
		}
		input := strings.NewReader(strings.Join(lines, "\n"))

		ch := make(chan RequestWithContext, 100)
		go func() {
			for range ch {
			}
		}()

		SendRequests(ch, input, "POST", headers)
		close(ch)
	}
}
