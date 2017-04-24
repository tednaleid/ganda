package base

import (
	"bufio"
	"fmt"
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

func NewSettings() *Settings {
	settings := Settings{
		RequestMethod:         "GET",
		Silent:                false,
		NoColor:               false,
		RequestWorkers:        30,
		SubdirLength:          0,
		ConnectTimeoutSeconds: 10,
		Retries:               0,
	}

	return &settings
}

type LeveledLogger struct {
	showColor bool
	silent    bool
	logger    *log.Logger
}

func NewSilentLogger(logger *log.Logger) *LeveledLogger {
	return &LeveledLogger{
		silent:    true,
		showColor: false,
		logger:    logger,
	}
}

func NewPlainLeveledLogger(logger *log.Logger) *LeveledLogger {
	return &LeveledLogger{
		silent:    false,
		showColor: false,
		logger:    logger,
	}
}

func NewLeveledLogger(logger *log.Logger) *LeveledLogger {
	return &LeveledLogger{
		silent:    false,
		showColor: true,
		logger:    logger,
	}
}

func (l *LeveledLogger) Info(format string, args ...interface{}) {
	if !l.silent {
		l.logger.Printf(format, args...)
	}
}

func (l *LeveledLogger) Warn(format string, args ...interface{}) {
	if l.showColor {
		l.logger.Printf("\033[31m"+format+"\033[0m", args...)
	} else if !l.silent {
		l.logger.Printf(format, args...)
	}
}

func (l *LeveledLogger) Success(format string, args ...interface{}) {
	if l.showColor {
		l.logger.Printf("\033[32m"+format+"\033[0m", args...)
	} else if !l.silent {
		l.logger.Printf(format, args...)
	}
}

func (l *LeveledLogger) LogResponse(statusCode int, message string) {
	if statusCode < 400 {
		l.Success("Response: %d %s", statusCode, message)
	} else {
		l.Warn("Response: %d %s", statusCode, message)
	}
}

func (l *LeveledLogger) LogError(err error, message string) {
	l.Warn("%s Error: %s", message, err)
}

type Context struct {
	RequestMethod          string
	WriteFiles             bool
	BaseDirectory          string
	SubdirLength           int
	RequestWorkers         int
	ConnectTimeoutDuration time.Duration
	Retries                int
	Logger                 *LeveledLogger
	Out                    *log.Logger
	RequestHeaders         []RequestHeader
	UrlScanner             *bufio.Scanner
}

func NewContext(settings *Settings) (*Context, error) {
	var err error

	context := Context{
		ConnectTimeoutDuration: time.Duration(settings.ConnectTimeoutSeconds) * time.Second,
		RequestMethod:          settings.RequestMethod,
		BaseDirectory:          settings.BaseDirectory,
		SubdirLength:           settings.SubdirLength,
		RequestWorkers:         settings.RequestWorkers,
		RequestHeaders:         settings.RequestHeaders,
		Out:                    log.New(os.Stdout, "", 0),
		Logger:                 createLeveledLogger(settings),
	}

	context.UrlScanner, err = UrlScanner(settings.UrlFilename, context.Logger)

	if len(settings.BaseDirectory) > 0 {
		context.WriteFiles = true
	} else {
		context.WriteFiles = false
	}

	return &context, err
}

func createLeveledLogger(settings *Settings) *LeveledLogger {
	stdErrLogger := log.New(os.Stderr, "", 0)

	if settings.Silent {
		stdErrLogger.SetOutput(ioutil.Discard)
		return NewSilentLogger(stdErrLogger)
	}

	if settings.NoColor {
		return NewPlainLeveledLogger(stdErrLogger)
	}

	return NewLeveledLogger(stdErrLogger)
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

func UrlScanner(urlFilename string, logger *LeveledLogger) (*bufio.Scanner, error) {
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
