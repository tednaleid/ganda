package execcontext

import (
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
	BaseDirectory          string
	BaseRetryDelayDuration time.Duration
	ConnectTimeoutDuration time.Duration
	DataTemplate           string
	DiscardBody            bool
	HashBody               bool
	In                     io.Reader
	Insecure               bool
	JsonEnvelope           bool
	Logger                 *logger.LeveledLogger
	Out                    io.Writer
	RequestHeaders         []config.RequestHeader
	RequestMethod          string
	RequestWorkers         int
	ResponseWorkers        int
	Retries                int64
	SubdirLength           int64
	ThrottlePerSecond      int64
	WriteFiles             bool
}

func New(conf *config.Config, in io.Reader, stderr io.Writer, stdout io.Writer) (*Context, error) {
	var err error

	context := Context{
		BaseDirectory:          conf.BaseDirectory,
		BaseRetryDelayDuration: time.Duration(conf.BaseRetryDelayMillis) * time.Millisecond,
		ConnectTimeoutDuration: time.Duration(conf.ConnectTimeoutMillis) * time.Millisecond,
		DataTemplate:           conf.DataTemplate,
		DiscardBody:            conf.DiscardBody,
		HashBody:               conf.HashBody,
		In:                     in,
		Insecure:               conf.Insecure,
		JsonEnvelope:           conf.JsonEnvelope,
		Logger:                 createLeveledLogger(conf, stderr),
		Out:                    stdout,
		RequestMethod:          conf.RequestMethod,
		RequestWorkers:         conf.RequestWorkers,
		RequestHeaders:         conf.RequestHeaders,
		ResponseWorkers:        conf.ResponseWorkers,
		Retries:                conf.Retries,
		SubdirLength:           conf.SubdirLength,
		ThrottlePerSecond:      math.MaxInt32,
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

	if len(conf.RequestFilename) > 0 {
		// replace stdin with the file
		context.In, err = requestFileReader(conf.RequestFilename)
	}

	if len(conf.BaseDirectory) > 0 {
		context.WriteFiles = true
	} else {
		context.WriteFiles = false
	}

	return &context, err
}

func createLeveledLogger(conf *config.Config, stderr io.Writer) *logger.LeveledLogger {

	if conf.Silent {
		return logger.NewSilentLogger()
	}

	stdErrLogger := log.New(stderr, "", 0)

	if conf.Color {
		return logger.NewLeveledLogger(stdErrLogger)
	}

	return logger.NewPlainLeveledLogger(stdErrLogger)
}

func requestFileReader(requestFilename string) (io.Reader, error) {
	if _, err := os.Stat(requestFilename); os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to open specified file: %s", requestFilename)
	}

	return os.Open(requestFilename)
}
