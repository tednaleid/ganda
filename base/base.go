package base

import (
	"log"
	"os"
	"strings"
	"time"
)

var Logger = log.New(os.Stderr, "", 0)
var Out = log.New(os.Stdout, "", 0)

type RequestHeader struct {
	Key   string
	Value string
}

type Config struct {
	WriteFiles             bool
	BaseDirectory          string
	RequestWorkers         int
	RequestMethod          string
	ConnectTimeoutDuration time.Duration
	UrlFilename            string
	RequestHeaders         []RequestHeader
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func StringToHeader(headerString string) RequestHeader {
	parts := strings.SplitN(headerString, ":", 2)
	return RequestHeader{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])}
}
