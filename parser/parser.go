package parser

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/tednaleid/ganda/config"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// Each line is an URL and optionally some TSV context that can be passed through
// an emitted along with the response output
func SendUrlsRequests(
	requestsWithContext chan<- RequestWithContext,
	reader *bufio.Reader,
	requestMethod string,
	staticHeaders []config.RequestHeader,
) error {
	tsvReader := csv.NewReader(reader)
	tsvReader.Comma = '\t'
	tsvReader.FieldsPerRecord = -1

	for {
		record, err := tsvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if len(record) > 0 {
			url := record[0]
			request := createRequest(url, nil, requestMethod, staticHeaders)
			recordContext := record[1:]

			if len(recordContext) == 0 {
				recordContext = nil
			}

			requestsWithContext <- RequestWithContext{Request: request, RequestContext: recordContext}
		}
	}
	return nil
}

type JsonLine struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Context  interface{}       `json:"context"`
	Headers  map[string]string `json:"headers"`
	Body     json.RawMessage   `json:"body"`
	BodyType string            `json:"bodyType"`
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

		body, err := parseBody(jsonLine.BodyType, jsonLine.Body)
		if err != nil {
			return fmt.Errorf("failed to parse body: %s", err)
		}

		// allow overriding of the request method per JSON line, but otherwise use the default
		method := requestMethod
		if jsonLine.Method != "" {
			method = jsonLine.Method
		}

		mergedHeaders := mergeHeaders(staticHeaders, jsonLine.Headers)

		request := createRequest(jsonLine.URL, body, method, mergedHeaders)
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

func parseBody(bodyType string, body json.RawMessage) (io.ReadCloser, error) {
	switch bodyType {
	case "escaped":
		str, err := strconv.Unquote(string(body))
		if err != nil {
			return nil, err
		}
		return io.NopCloser(strings.NewReader(str)), nil
	case "base64":
		unquoted, err := strconv.Unquote(string(body))
		data, err := base64.StdEncoding.DecodeString(unquoted)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(data)), nil
	case "json", "":
		// Use the JSON as is
		return io.NopCloser(bytes.NewReader(body)), nil
	default:
		return nil, fmt.Errorf("unsupported body type: %s, valid values: \"json\", \"base64\", \"escaped\"", bodyType)
	}
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

	request.Header.Add("connection", "keep-alive")

	for _, header := range requestHeaders {
		request.Header.Add(header.Key, header.Value)
	}

	return request
}
