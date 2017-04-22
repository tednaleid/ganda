package requests

import (
	"crypto/tls"
	"github.com/tednaleid/ganda/base"
	"net/http"
	"sync"
)

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
	client := httpClient(context)
	for url := range requests {
		request := createRequest(url, context.RequestMethod, context.RequestHeaders)

		response, err := client.Do(request)

		if err == nil {
			responses <- response
		} else {
			context.Logger.Println(url, "Error:", err)
		}
	}
}

func httpClient(context *base.Context) *http.Client {
	return &http.Client{
		Timeout: context.ConnectTimeoutDuration,
		Transport: &http.Transport{
			MaxIdleConns:        500,
			MaxIdleConnsPerHost: 50,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
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
