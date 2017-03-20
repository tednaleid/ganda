package requests

import (
	"crypto/tls"
	"github.com/tednaleid/ganda/base"
	"net/http"
	"sync"
)

func StartRequestWorkers(requests <-chan string, responses chan<- *http.Response, config base.Config) *sync.WaitGroup {
	var requestWaitGroup sync.WaitGroup
	requestWaitGroup.Add(config.RequestWorkers)

	for i := 1; i <= config.RequestWorkers; i++ {
		go func() {
			requestWorker(config, requests, responses)
			requestWaitGroup.Done()
		}()
	}

	return &requestWaitGroup
}

func requestWorker(config base.Config, requests <-chan string, responses chan<- *http.Response) {
	client := httpClient(config)
	for url := range requests {
		request := createRequest(url, config.RequestMethod, config.RequestHeaders)

		response, err := client.Do(request)

		if err == nil {
			responses <- response
		} else {
			base.Logger.Println(url, "Error:", err)
		}
	}
}

func httpClient(config base.Config) *http.Client {
	return &http.Client{
		Timeout: config.ConnectTimeoutDuration,
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
