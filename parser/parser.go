package parser

import (
	"bufio"
	"encoding/csv"
	"github.com/tednaleid/ganda/config"
	"io"
	"net/http"
)

type InputType int

const (
	Unknown InputType = iota
	Urls
	JsonLines
)

type RequestWithContext struct {
	Request        *http.Request
	RequestContext interface{}
}

func SendRequests(
	requestsWithContext chan<- RequestWithContext,
	in io.Reader,
	requestMethod string,
	staticHeaders []config.RequestHeader,
) {
	reader := bufio.NewReader(in)
	inputType, _ := determineInputType(reader)

	if inputType == Urls {
		SendUrlsRequests(requestsWithContext, reader, requestMethod, staticHeaders)
	} else if inputType == JsonLines {
		SendJsonLinesRequests(requestsWithContext, reader, requestMethod, staticHeaders)
	}
}

// Each line is an URL and optionally some CSV context that can be passed through
// an emitted along with the response output
func SendUrlsRequests(
	requestsWithContext chan<- RequestWithContext,
	reader *bufio.Reader,
	requestMethod string,
	staticHeaders []config.RequestHeader,
) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = '\t'
	csvReader.FieldsPerRecord = -1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			// TODO handle more gracefully when CSV isn't well formed
			panic(err)
		}

		if len(record) > 0 {
			url := record[0]
			request := createRequest(url, nil, requestMethod, staticHeaders)
			requestsWithContext <- RequestWithContext{Request: request, RequestContext: record[1:]}
		}
	}
}

func SendJsonLinesRequests(
	requestsWithContext chan<- RequestWithContext,
	reader *bufio.Reader,
	requestMethod string,
	staticHeaders []config.RequestHeader,
) {
	// TODO

}

// current assumption is that the first character is '{' for a stream of json lines,
// otherwise it's a stream of urls
func determineInputType(bufferedReader *bufio.Reader) (InputType, error) {
	initialByte, err := bufferedReader.Peek(1)

	if err != nil {
		return Unknown, err
	}

	if initialByte[0] == '{' {
		return JsonLines, nil
	}

	return Urls, nil
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
