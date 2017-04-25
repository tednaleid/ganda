package requests

import (
	"fmt"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"net/http"
	"sync"
	"time"
)

func StartRequestWorkers(requests <-chan string, responses chan<- *http.Response, context *execcontext.Context) *sync.WaitGroup {
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

func requestWorker(context *execcontext.Context, requests <-chan string, responses chan<- *http.Response) {
	httpClient := context.NewHttpClient()
	for url := range requests {
		request := createRequest(url, context.RequestMethod, context.RequestHeaders)

		finalResponse, err := requestWithRetry(httpClient, request, 0)

		if err == nil {
			responses <- finalResponse
		} else {
			httpClient.Logger.LogError(err, url)
		}
	}
}

func requestWithRetry(httpClient *execcontext.HttpClient, request *http.Request, previouslyFailed int) (*http.Response, error) {
	response, err := httpClient.Client.Do(request)

	if previouslyFailed < httpClient.MaxRetries && (err != nil || response.StatusCode >= 500) {
		failed := previouslyFailed + 1

		message := fmt.Sprintf("%s (%d)", request.URL.String(), failed)

		if err == nil {
			httpClient.Logger.LogResponse(response.StatusCode, message)
		} else {
			httpClient.Logger.LogError(err, message)
		}

		time.Sleep(time.Duration(failed) * time.Second)

		return requestWithRetry(httpClient, request, failed)
	}

	return response, err
}

func createRequest(url string, requestMethod string, requestHeaders []config.RequestHeader) *http.Request {
	request, err := http.NewRequest(requestMethod, url, nil)

	if err != nil {
		panic(err)
	}

	for _, header := range requestHeaders {
		request.Header.Add(header.Key, header.Value)
	}

	request.Header.Add("connection", "keep-alive")
	return request
}
