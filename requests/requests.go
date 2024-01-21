package requests

import (
	"crypto/tls"
	"fmt"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/logger"
	"github.com/tednaleid/ganda/parser"
	"github.com/tednaleid/ganda/responses"
	"math"
	"net/http"
	"sync"
	"time"
)

type HttpClient struct {
	MaxRetries int64
	Client     *http.Client
	Logger     *logger.LeveledLogger
}

func NewHttpClient(context *execcontext.Context) *HttpClient {
	return &HttpClient{
		MaxRetries: context.Retries,
		Logger:     context.Logger,
		Client: &http.Client{
			Timeout: context.ConnectTimeoutDuration,
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 50,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: context.Insecure,
				},
			},
		},
	}
}

func StartRequestWorkers(
	requestsWithContext <-chan parser.RequestWithContext,
	responsesWithContext chan<- *responses.ResponseWithContext,
	context *execcontext.Context,
) *sync.WaitGroup {
	var requestWaitGroup sync.WaitGroup
	requestWaitGroup.Add(context.RequestWorkers)

	var rateLimitTicker *time.Ticker

	// don't throttle if we're not limiting the number of requests per second
	if context.ThrottlePerSecond != math.MaxInt32 {
		rateLimitTicker = time.NewTicker(time.Second / time.Duration(context.ThrottlePerSecond))
		defer rateLimitTicker.Stop()
	}

	for i := 1; i <= context.RequestWorkers; i++ {
		go func() {
			requestWorker(context, requestsWithContext, responsesWithContext, rateLimitTicker)
			requestWaitGroup.Done()
		}()
	}

	return &requestWaitGroup
}

func requestWorker(
	context *execcontext.Context,
	requestsWithContext <-chan parser.RequestWithContext,
	responsesWithContext chan<- *responses.ResponseWithContext,
	rateLimitTicker *time.Ticker,
) {
	httpClient := NewHttpClient(context)

	for requestWithContext := range requestsWithContext {
		if rateLimitTicker != nil {
			<-rateLimitTicker.C // wait for the next tick to send the request
		}

		finalResponse, err := requestWithRetry(httpClient, requestWithContext, context.BaseRetryDelayDuration)

		if err == nil {
			responsesWithContext <- finalResponse
		}
	}
}

func requestWithRetry(
	httpClient *HttpClient,
	requestWithContext parser.RequestWithContext,
	baseRetryDelay time.Duration,
) (*responses.ResponseWithContext, error) {
	var response *http.Response
	var err error

	for attempts := int64(1); ; attempts++ {
		response, err = httpClient.Client.Do(requestWithContext.Request)

		responseWithContext := &responses.ResponseWithContext{
			Response:       response,
			RequestContext: requestWithContext.RequestContext,
		}

		if err == nil && response.StatusCode < 500 {
			// return successful response or non-server error, we don't retry those
			return responseWithContext, nil
		}

		message := requestWithContext.Request.URL.String()

		if err == nil {
			httpClient.Logger.LogResponse(response.StatusCode, message)
		} else {
			httpClient.Logger.LogError(err, message)
		}

		if attempts > httpClient.MaxRetries {
			return responseWithContext, fmt.Errorf("maximum number of retries (%d) reached for request", httpClient.MaxRetries)
		}

		time.Sleep(baseRetryDelay * time.Duration(2^attempts))
	}

}
