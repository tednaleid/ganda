package logger

import "log"

type LeveledLogger struct {
	showColor bool
	silent    bool
	logger    *log.Logger
}

func NewSilentLogger() *LeveledLogger {
	return &LeveledLogger{
		silent:    true,
		showColor: false,
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
