package config

import (
	"math"
	"strings"
)

type Config struct {
	Silent                bool
	Insecure              bool
	NoColor               bool
	BaseDirectory         string
	RequestWorkers        int
	ResponseWorkers       int
	SubdirLength          int
	RequestMethod         string
	ConnectTimeoutSeconds int
	ThrottlePerSecond     int
	Retries               int
	RequestHeaders        []RequestHeader
	UrlFilename           string
}

func New() *Config {
	return &Config{
		RequestMethod:         "GET",
		Insecure:              false,
		Silent:                false,
		NoColor:               false,
		RequestWorkers:        30,
		SubdirLength:          0,
		ConnectTimeoutSeconds: 10,
		ThrottlePerSecond:     math.MaxInt32,
		Retries:               0,
	}
}

type RequestHeader struct {
	Key   string
	Value string
}

func NewRequestHeader(headerString string) RequestHeader {
	parts := strings.SplitN(headerString, ":", 2)
	return RequestHeader{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])}
}
