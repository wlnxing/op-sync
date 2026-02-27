package openlistsync

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelError
)

func ParseLogLevel(s string) (LogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LogLevelDebug, nil
	case "info":
		return LogLevelInfo, nil
	case "error", "":
		return LogLevelError, nil
	default:
		return LogLevelError, fmt.Errorf("invalid log level: %s (allowed: debug, info, error)", s)
	}
}

type Logger struct {
	base  *log.Logger
	level LogLevel
}

func NewLogger(out io.Writer, level LogLevel) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{
		base:  log.New(out, "", log.LstdFlags),
		level: level,
	}
}

func (l *Logger) Debugf(format string, args ...any) {
	l.logf(LogLevelDebug, "DEBUG", format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.logf(LogLevelInfo, "INFO", format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.logf(LogLevelError, "ERROR", format, args...)
}

func (l *Logger) logf(level LogLevel, label, format string, args ...any) {
	if l == nil || l.base == nil {
		return
	}
	if level < l.level {
		return
	}
	l.base.Printf("[%s] %s", label, fmt.Sprintf(format, args...))
}
