package config

import (
	"errors"
	"math"
	"strings"
)

type Config struct {
	BaseDirectory        string
	BaseRetryDelayMillis int64
	Color                bool
	ConnectTimeoutMillis int64
	DataTemplate         string
	DiscardBody          bool
	HashBody             bool
	Insecure             bool
	JsonEnvelope         bool
	RequestFilename      string
	RequestHeaders       []RequestHeader
	RequestMethod        string
	RequestWorkers       int
	ResponseWorkers      int
	Retries              int64
	Silent               bool
	SubdirLength         int64
	ThrottlePerSecond    int64
}

func New() *Config {
	return &Config{
		BaseRetryDelayMillis: 1_000,
		Color:                false,
		ConnectTimeoutMillis: 10_000,
		DataTemplate:         "",
		DiscardBody:          false,
		HashBody:             false,
		Insecure:             false,
		JsonEnvelope:         false,
		RequestMethod:        "GET",
		RequestWorkers:       1,
		Retries:              0,
		Silent:               false,
		SubdirLength:         0,
		ThrottlePerSecond:    math.MaxInt32,
	}
}

type RequestHeader struct {
	Key   string
	Value string
}

func NewRequestHeader(headerString string) (RequestHeader, error) {
	if strings.Contains(headerString, ":") {
		parts := strings.SplitN(headerString, ":", 2)
		return RequestHeader{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])}, nil
	}

	return RequestHeader{}, errors.New("Header should be in the format 'Key: value', missing ':' -> " + headerString)
}

func ConvertRequestHeaders(stringHeaders []string) ([]RequestHeader, error) {
	var requestHeaders []RequestHeader

	for _, header := range stringHeaders {
		var requestHeader RequestHeader
		requestHeader, err := NewRequestHeader(header)

		if err != nil {
			return requestHeaders, err
		}

		requestHeaders = append(requestHeaders, requestHeader)
	}

	return requestHeaders, nil
}
