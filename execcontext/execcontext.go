package execcontext

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/logger"
	"log"
	"net/http"
	"os"
	"time"
)

type Context struct {
	RequestMethod          string
	WriteFiles             bool
	BaseDirectory          string
	SubdirLength           int
	RequestWorkers         int
	ConnectTimeoutDuration time.Duration
	Retries                int
	Logger                 *logger.LeveledLogger
	Out                    *log.Logger
	RequestHeaders         []config.RequestHeader
	UrlScanner             *bufio.Scanner
}

func New(conf *config.Config) (*Context, error) {
	var err error

	context := Context{
		ConnectTimeoutDuration: time.Duration(conf.ConnectTimeoutSeconds) * time.Second,
		RequestMethod:          conf.RequestMethod,
		BaseDirectory:          conf.BaseDirectory,
		SubdirLength:           conf.SubdirLength,
		RequestWorkers:         conf.RequestWorkers,
		RequestHeaders:         conf.RequestHeaders,
		Out:                    log.New(os.Stdout, "", 0),
		Logger:                 createLeveledLogger(conf),
	}

	context.UrlScanner, err = createUrlScanner(conf.UrlFilename, context.Logger)

	if len(conf.BaseDirectory) > 0 {
		context.WriteFiles = true
	} else {
		context.WriteFiles = false
	}

	return &context, err
}

func createLeveledLogger(conf *config.Config) *logger.LeveledLogger {

	if conf.Silent {
		return logger.NewSilentLogger()
	}

	stdErrLogger := log.New(os.Stderr, "", 0)

	if conf.NoColor {
		return logger.NewPlainLeveledLogger(stdErrLogger)
	}

	return logger.NewLeveledLogger(stdErrLogger)
}

func createUrlScanner(urlFilename string, logger *logger.LeveledLogger) (*bufio.Scanner, error) {
	if len(urlFilename) > 0 {
		logger.Info("Opening file of urls at: %s", urlFilename)
		return urlFileScanner(urlFilename)
	}
	return urlStdinScanner(), nil
}

func urlStdinScanner() *bufio.Scanner {
	return bufio.NewScanner(os.Stdin)
}

func urlFileScanner(urlFilename string) (*bufio.Scanner, error) {
	if _, err := os.Stat(urlFilename); os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to open specified file: %s", urlFilename)
	}

	file, err := os.Open(urlFilename)
	return bufio.NewScanner(file), err
}

type HttpClient struct {
	MaxRetries int
	Client     *http.Client
	Logger     *logger.LeveledLogger
}

func (context *Context) NewHttpClient() *HttpClient {
	return &HttpClient{
		MaxRetries: context.Retries,
		Logger:     context.Logger,
		Client: &http.Client{
			Timeout: context.ConnectTimeoutDuration,
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 50,
				TLSClientConfig: &tls.Config{
					// TODO turn this into a -k flag
					InsecureSkipVerify: true,
				},
			},
		},
	}
}
