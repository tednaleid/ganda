package parser

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
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
) error {
	reader := bufio.NewReader(in)
	inputType, _ := determineInputType(reader)

	if inputType == JsonLines {
		return SendJsonLinesRequests(requestsWithContext, reader, requestMethod, staticHeaders)
	}

	return SendUrlsRequests(requestsWithContext, reader, requestMethod, staticHeaders)
}

// Each line is an URL and optionally some CSV context that can be passed through
// an emitted along with the response output
func SendUrlsRequests(
	requestsWithContext chan<- RequestWithContext,
	reader *bufio.Reader,
	requestMethod string,
	staticHeaders []config.RequestHeader,
) error {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = '\t'
	csvReader.FieldsPerRecord = -1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if len(record) > 0 {
			url := record[0]
			request := createRequest(url, nil, requestMethod, staticHeaders)
			requestsWithContext <- RequestWithContext{Request: request, RequestContext: record[1:]}
		}
	}
	return nil
}

type JsonLine struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Context interface{}       `json:"context"`
	Headers map[string]string `json:"headers"`
}

func SendJsonLinesRequests(
	requestsWithContext chan<- RequestWithContext,
	reader *bufio.Reader,
	requestMethod string,
	staticHeaders []config.RequestHeader,
) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		var jsonLine JsonLine

		err := json.Unmarshal([]byte(line), &jsonLine)
		if err != nil {
			return fmt.Errorf("%s: %s", err.Error(), line)
		} else if jsonLine.URL == "" {
			return fmt.Errorf("missing url property: %s", line)
		}

		// allow overriding of the request method per JSON line, but otherwise use the default
		method := requestMethod
		if jsonLine.Method != "" {
			method = jsonLine.Method
		}

		mergedHeaders := mergeHeaders(staticHeaders, jsonLine.Headers)

		request := createRequest(jsonLine.URL, nil, method, mergedHeaders)
		requestsWithContext <- RequestWithContext{Request: request, RequestContext: jsonLine.Context}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func mergeHeaders(staticHeaders []config.RequestHeader, jsonLineHeaders map[string]string) []config.RequestHeader {
	if len(jsonLineHeaders) == 0 {
		return staticHeaders
	}

	headersMap := make(map[string]string)
	for _, header := range staticHeaders {
		headersMap[header.Key] = header.Value
	}

	for key, value := range jsonLineHeaders {
		headersMap[key] = value
	}

	mergedHeaders := make([]config.RequestHeader, 0, len(headersMap))
	for key, value := range headersMap {
		mergedHeaders = append(mergedHeaders, config.RequestHeader{Key: key, Value: value})
	}

	return mergedHeaders
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
