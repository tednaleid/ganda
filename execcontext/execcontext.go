package execcontext

import (
	"bufio"
	"fmt"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/logger"
	"io"
	"log"
	"math"
	"os"
	"time"
)

type Context struct {
	RequestMethod          string
	WriteFiles             bool
	JsonEnvelope           bool
	HashBody               bool
	DiscardBody            bool
	Insecure               bool
	BaseDirectory          string
	DataTemplate           string
	SubdirLength           int64
	RequestWorkers         int
	ResponseWorkers        int
	ConnectTimeoutDuration time.Duration
	ThrottlePerSecond      int64
	Retries                int64
	Logger                 *logger.LeveledLogger
	Out                    io.Writer
	RequestHeaders         []config.RequestHeader
	RequestScanner         *bufio.Scanner
}

func New(conf *config.Config) (*Context, error) {
	var err error

	context := Context{
		ConnectTimeoutDuration: time.Duration(conf.ConnectTimeoutSeconds) * time.Second,
		Insecure:               conf.Insecure,
		JsonEnvelope:           conf.JsonEnvelope,
		HashBody:               conf.HashBody,
		DiscardBody:            conf.DiscardBody,
		RequestMethod:          conf.RequestMethod,
		BaseDirectory:          conf.BaseDirectory,
		DataTemplate:           conf.DataTemplate,
		SubdirLength:           conf.SubdirLength,
		RequestWorkers:         conf.RequestWorkers,
		ResponseWorkers:        conf.ResponseWorkers,
		RequestHeaders:         conf.RequestHeaders,
		ThrottlePerSecond:      math.MaxInt32,
		Out:                    os.Stdout,
		Logger:                 createLeveledLogger(conf),
	}

	if conf.ThrottlePerSecond > 0 {
		context.ThrottlePerSecond = conf.ThrottlePerSecond
	}

	if context.RequestWorkers <= 0 {
		context.RequestWorkers = 1
	}

	if context.ResponseWorkers <= 0 {
		context.ResponseWorkers = context.RequestWorkers
	}

	context.RequestScanner, err = createRequestScanner(conf.RequestFilename, context.Logger)

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

func createRequestScanner(requestFilename string, logger *logger.LeveledLogger) (*bufio.Scanner, error) {
	if len(requestFilename) > 0 {
		logger.Info("Opening file of requests at: %s", requestFilename)
		return requestFileScanner(requestFilename)
	}
	return urlStdinScanner(), nil
}

const MaxTokenSize = 1024 * 1024 * 1024

func urlStdinScanner() *bufio.Scanner {
	s := bufio.NewScanner(os.Stdin)
	s.Buffer(make([]byte, 1024*2), MaxTokenSize)
	return s
}

func requestFileScanner(requestFilename string) (*bufio.Scanner, error) {
	if _, err := os.Stat(requestFilename); os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to open specified file: %s", requestFilename)
	}

	file, err := os.Open(requestFilename)
	return bufio.NewScanner(file), err
}
