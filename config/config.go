package config

import "strings"

type Config struct {
	Silent                bool
	NoColor               bool
	BaseDirectory         string
	RequestWorkers        int
	SubdirLength          int
	RequestMethod         string
	ConnectTimeoutSeconds int
	Retries               int
	RequestHeaders        []RequestHeader
	UrlFilename           string
}

func New() *Config {
	return &Config{
		RequestMethod:         "GET",
		Silent:                false,
		NoColor:               false,
		RequestWorkers:        30,
		SubdirLength:          0,
		ConnectTimeoutSeconds: 10,
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
