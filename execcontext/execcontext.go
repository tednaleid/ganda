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
	In                     io.Reader
	Out                    io.Writer
	RequestHeaders         []config.RequestHeader
}

func New(conf *config.Config, in io.Reader, stderr io.Writer, stdout io.Writer) (*Context, error) {
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
		In:                     in,
		Out:                    stdout,
		Logger:                 createLeveledLogger(conf, stderr),
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
