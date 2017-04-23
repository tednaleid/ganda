package base

import (
	"bufio"
	"github.com/tednaleid/ganda/urls"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

type RequestHeader struct {
	Key   string
	Value string
}

type Settings struct {
	Silent                bool
	BaseDirectory         string
	RequestWorkers        int
	SubdirLength          int
	RequestMethod         string
	ConnectTimeoutSeconds int
	Retries               int
	RequestHeaders        []RequestHeader
	UrlFilename           string
}

func NewSettings() *Settings {
	settings := Settings{
		RequestMethod:         "GET",
		Silent:                false,
		RequestWorkers:        30,
		SubdirLength:          0,
		ConnectTimeoutSeconds: 10,
		Retries:               0,
	}

	return &settings
}

type Context struct {
	RequestMethod          string
	WriteFiles             bool
	BaseDirectory          string
	SubdirLength           int
	RequestWorkers         int
	ConnectTimeoutDuration time.Duration
	Retries                int
	Logger                 *log.Logger
	Out                    *log.Logger
	RequestHeaders         []RequestHeader
	UrlScanner             *bufio.Scanner
}

func NewContext(settings *Settings) (*Context, error) {
	var err error

	context := Context{
		RequestMethod:          settings.RequestMethod,
		BaseDirectory:          settings.BaseDirectory,
		SubdirLength:           settings.SubdirLength,
		RequestWorkers:         settings.RequestWorkers,
		RequestHeaders:         settings.RequestHeaders,
		ConnectTimeoutDuration: time.Duration(settings.ConnectTimeoutSeconds) * time.Second,
		Out:    log.New(os.Stdout, "", 0),
		Logger: log.New(os.Stderr, "", 0),
	}

	context.UrlScanner, err = urls.UrlScanner(settings.UrlFilename, context.Logger)

	if len(settings.BaseDirectory) > 0 {
		context.WriteFiles = true
	} else {
		context.WriteFiles = false
	}

	if settings.Silent {
		context.Logger.SetOutput(ioutil.Discard)
	}

	return &context, err
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
