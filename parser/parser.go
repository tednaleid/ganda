package parser

import (
	"fmt"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"io"
	"net/http"
	"strings"
	"time"
)

func SendRequests(context *execcontext.Context, requests chan<- *http.Request) {
	var body io.Reader = nil
	var url string
	requestScanner := context.RequestScanner
	throttleRequestsPerSecond := context.ThrottlePerSecond
	count := int64(0)
	throttle := time.Tick(time.Second)

	for requestScanner.Scan() {
		count++
		if count%throttleRequestsPerSecond == 0 {
			<-throttle
		}

		if context.DataTemplate == "" {
			var text = requestScanner.Text()
			url, body = ParseUrlAndOptionalBody(text)
		} else {
			url, body = ParseTemplatedInput(requestScanner.Text(), context.DataTemplate)
		}

		request := createRequest(url, body, context.RequestMethod, context.RequestHeaders)
		requests <- request
	}
}

func ParseUrlAndOptionalBody(input string) (string, io.Reader) {
	tokens := strings.SplitN(input, " ", 2)

	url := tokens[0]

	if len(tokens) == 1 {
		return url, nil // no body, just an url
	}

	return url, strings.NewReader(tokens[1])
}

// input string should be an url followed by space delimited values that will be passed to the Sprintf function
// where it will replace the "%s" with strings
func ParseTemplatedInput(input string, dataTemplate string) (string, io.Reader) {
	tokens := strings.Split(input, " ")

	url := tokens[0]

	if len(tokens) == 1 {
		return url, strings.NewReader(dataTemplate) // just an url, static body
	}

	// Sprintf wants a []interface{} which isn't compatible with a []string; see: https://golang.org/doc/faq#convert_slice_of_interface
	bodyTokens := make([]interface{}, len(tokens)-1)
	for i, value := range tokens[1:] {
		bodyTokens[i] = value
	}

	return url, strings.NewReader(fmt.Sprintf(dataTemplate, bodyTokens...))
}

func createRequest(url string, body io.Reader, requestMethod string, requestHeaders []config.RequestHeader) *http.Request {
	request, err := http.NewRequest(requestMethod, url, body)

	if err != nil {
		panic(err)
	}

	for _, header := range requestHeaders {
		request.Header.Add(header.Key, header.Value)
	}

	request.Header.Add("connection", "keep-alive")
	return request
}
