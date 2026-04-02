package requests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/parser"
	"github.com/tednaleid/ganda/responses"
)

func newTestContext(retries int) *execcontext.Context {
	conf := config.New()
	conf.Retries = retries
	conf.Silent = true
	ctx, _ := execcontext.New(conf, strings.NewReader(""), io.Discard, io.Discard)
	return ctx
}

func TestNewHttpClient(t *testing.T) {
	ctx := newTestContext(3)
	client := NewHttpClient(ctx)

	assert.Equal(t, 3, client.MaxRetries)
	assert.NotNil(t, client.Client)
	assert.NotNil(t, client.Logger)
}

func TestRequestWithRetrySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	ctx := newTestContext(0)
	client := NewHttpClient(ctx)

	req, _ := http.NewRequest("GET", server.URL, nil)
	rwc := parser.RequestWithContext{Request: req}

	resp, err := requestWithRetry(client, rwc, time.Millisecond)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Response.StatusCode)
	body, _ := io.ReadAll(resp.Response.Body)
	resp.Response.Body.Close()
	assert.Equal(t, "ok", string(body))
}

func TestRequestWithRetryReturns4xxWithoutRetrying(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ctx := newTestContext(3)
	client := NewHttpClient(ctx)

	req, _ := http.NewRequest("GET", server.URL, nil)
	rwc := parser.RequestWithContext{Request: req}

	resp, err := requestWithRetry(client, rwc, time.Millisecond)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.Response.StatusCode)
	assert.Equal(t, 1, callCount, "4xx should not trigger retries")
	resp.Response.Body.Close()
}

func TestRequestWithRetryRetries5xx(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("recovered"))
	}))
	defer server.Close()

	ctx := newTestContext(5)
	client := NewHttpClient(ctx)

	req, _ := http.NewRequest("GET", server.URL, nil)
	rwc := parser.RequestWithContext{Request: req}

	resp, err := requestWithRetry(client, rwc, time.Millisecond)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Response.StatusCode)
	assert.Equal(t, 3, callCount, "should have retried until success")
	body, _ := io.ReadAll(resp.Response.Body)
	resp.Response.Body.Close()
	assert.Equal(t, "recovered", string(body))
}

func TestRequestWithRetryExhaustsRetries(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := newTestContext(2)
	client := NewHttpClient(ctx)

	req, _ := http.NewRequest("GET", server.URL, nil)
	rwc := parser.RequestWithContext{Request: req}

	_, err := requestWithRetry(client, rwc, time.Millisecond)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of retries (2) reached")
	// attempts 1 (initial) + 2 retries = 3 total calls
	assert.Equal(t, 3, callCount)
}

func TestStartRequestWorkers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer server.Close()

	ctx := newTestContext(0)
	ctx.RequestWorkers = 2

	requestsChan := make(chan parser.RequestWithContext, 2)
	responsesChan := make(chan *responses.ResponseWithContext, 2)

	wg := StartRequestWorkers(requestsChan, responsesChan, nil, ctx)

	req1, _ := http.NewRequest("GET", server.URL+"/a", nil)
	req2, _ := http.NewRequest("GET", server.URL+"/b", nil)
	requestsChan <- parser.RequestWithContext{Request: req1}
	requestsChan <- parser.RequestWithContext{Request: req2}
	close(requestsChan)

	wg.Wait()
	close(responsesChan)

	var results []*responses.ResponseWithContext
	for r := range responsesChan {
		results = append(results, r)
	}

	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, http.StatusOK, r.Response.StatusCode)
		r.Response.Body.Close()
	}
}

func TestStartRequestWorkersWithRateLimiting(t *testing.T) {
	callCount := 0
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := newTestContext(0)
	ctx.RequestWorkers = 1

	requestsChan := make(chan parser.RequestWithContext, 3)
	responsesChan := make(chan *responses.ResponseWithContext, 3)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	wg := StartRequestWorkers(requestsChan, responsesChan, ticker, ctx)

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		requestsChan <- parser.RequestWithContext{Request: req}
	}
	close(requestsChan)

	wg.Wait()
	close(responsesChan)

	for r := range responsesChan {
		r.Response.Body.Close()
	}

	mu.Lock()
	assert.Equal(t, 3, callCount)
	mu.Unlock()
}
