package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type LogLevel int

const (
	LogLevelOff LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	SetLevel(level LogLevel)
}

type DefaultLogger struct {
	logger *log.Logger
	level  LogLevel
}

func NewLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stderr, "", log.LstdFlags),
		level:  level,
	}
}

func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *DefaultLogger) log(level LogLevel, msg string, keysAndValues ...any) {
	if level <= l.level {
		l.logger.Printf("%s: %s %v", level, msg, keysAndValues)
	}
}

func (l *DefaultLogger) Debug(msg string, keysAndValues ...any) {
	l.log(LogLevelDebug, msg, keysAndValues...)
}

func (l *DefaultLogger) Info(msg string, keysAndValues ...any) {
	l.log(LogLevelInfo, msg, keysAndValues...)
}

func (l *DefaultLogger) Warn(msg string, keysAndValues ...any) {
	l.log(LogLevelWarn, msg, keysAndValues...)
}

func (l *DefaultLogger) Error(msg string, keysAndValues ...any) {
	l.log(LogLevelError, msg, keysAndValues...)
}

func (l LogLevel) String() string {
	return [...]string{"OFF", "ERROR", "WARN", "INFO", "DEBUG"}[l]
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	switch strings.ToUpper(string(text)) {
	case "OFF":
		*l = LogLevelOff
	case "ERROR":
		*l = LogLevelError
	case "WARN":
		*l = LogLevelWarn
	case "INFO":
		*l = LogLevelInfo
	case "DEBUG":
		*l = LogLevelDebug
	default:
		return fmt.Errorf("invalid log level: %s", string(text))
	}
	return nil
}
