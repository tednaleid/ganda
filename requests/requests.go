package requests

import (
	"crypto/tls"
	"github.com/tednaleid/ganda/base"
	"log"
	"net/http"
	"sync"
	"time"
)

type HttpClient struct {
	MaxRetries int
	Client     *http.Client
	Logger     *log.Logger
}

func StartRequestWorkers(requests <-chan string, responses chan<- *http.Response, context *base.Context) *sync.WaitGroup {
	var requestWaitGroup sync.WaitGroup
	requestWaitGroup.Add(context.RequestWorkers)

	for i := 1; i <= context.RequestWorkers; i++ {
		go func() {
			requestWorker(context, requests, responses)
			requestWaitGroup.Done()
		}()
	}

	return &requestWaitGroup
}

func requestWorker(context *base.Context, requests <-chan string, responses chan<- *http.Response) {
	httpClient := NewHttpClient(context)
	for url := range requests {
		request := createRequest(url, context.RequestMethod, context.RequestHeaders)

		finalResponse, err := requestWithRetry(httpClient, request, 0)

		if err == nil {
			responses <- finalResponse
		} else {
			context.Logger.Println(url, "Error:", err)
		}
	}
}

func requestWithRetry(httpClient *HttpClient, request *http.Request, previouslyFailed int) (*http.Response, error) {
	response, err := httpClient.Client.Do(request)

	if previouslyFailed < httpClient.MaxRetries && (err != nil || response.StatusCode >= 500) {
		// TODO add some output on failure
		failed := previouslyFailed + 1
		time.Sleep(time.Duration(failed) * time.Second)
		return requestWithRetry(httpClient, request, failed)
	}

	return response, err
}

func NewHttpClient(context *base.Context) *HttpClient {
	return &HttpClient{
		MaxRetries: context.Retries,
		Logger:     context.Logger,
		Client: &http.Client{
			Timeout: context.ConnectTimeoutDuration,
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 50,
				TLSClientConfig: &tls.Config{
					// TODO turn this into a -k flag
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

func createRequest(url string, requestMethod string, requestHeaders []base.RequestHeader) *http.Request {
	request, err := http.NewRequest(requestMethod, url, nil)
	base.Check(err)

	for _, header := range requestHeaders {
		request.Header.Add(header.Key, header.Value)
	}

	request.Header.Add("connection", "keep-alive")
	return request
}
