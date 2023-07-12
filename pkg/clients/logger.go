package clients

import (
	"io"

	"github.com/charmbracelet/log"
	"github.com/hashicorp/go-hclog"
)

const (
	LogLevelInfo  = "info"
	LogLevelDebug = "debug"
	LogLevelTrace = "trace"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Logger defines a abstract logger that can be used to log to the output
type Logger interface {
	// Set the logger level
	SetLevel(level string)

	Level() string

	// Set the logger output
	SetOutput(w io.Writer)

	Output() io.Writer

	// Info logs to info level
	Info(message string, keyvals ...interface{})
	// Debug logs to debug level
	Debug(message string, keyvals ...interface{})
	// Error logs to error level
	Error(message string, keyvals ...interface{})
	// Warn logs to warn level
	Warn(message string, keyvals ...interface{})
	// Trace logs to trace level
	Trace(message string, keyvals ...interface{})

	StandardWriter() io.Writer

	IsInfo() bool
	IsDebug() bool
	IsError() bool
	IsTrace() bool
	IsWarn() bool
}

type CharmLogger struct {
	internal *log.Logger
	writer   io.Writer
	level    string
}

func NewLogger(w io.Writer, level string) Logger {
	l := log.New(w)
	l.SetLevel(log.ParseLevel(level))

	return &CharmLogger{l, w, level}
}

func (l *CharmLogger) SetOutput(w io.Writer) {
	l.writer = w
	l.internal.SetOutput(w)
}

func (l *CharmLogger) Output() io.Writer {
	return l.writer
}

func (l *CharmLogger) IsInfo() bool {
	return l.level == LogLevelInfo
}

func (l *CharmLogger) IsDebug() bool {
	return l.level == LogLevelDebug
}

func (l *CharmLogger) IsError() bool {
	return l.level == LogLevelError
}

func (l *CharmLogger) IsWarn() bool {
	return l.level == LogLevelWarn
}

func (l *CharmLogger) IsTrace() bool {
	return l.level == LogLevelTrace
}

func (l *CharmLogger) SetLevel(level string) {
	l.level = level
	l.internal.SetLevel(log.ParseLevel(level))
}

func (l *CharmLogger) Level() string {
	return l.level
}

func (l *CharmLogger) StandardWriter() io.Writer {
	return l.internal.StandardLog(log.StandardLogOptions{ForceLevel: log.DebugLevel}).Writer()
}

func (l *CharmLogger) Info(message string, keyvals ...interface{}) {
	l.internal.Info(message, keyvals...)
}

func (l *CharmLogger) Debug(message string, keyvals ...interface{}) {
	l.internal.Debug(message, keyvals...)
}

func (l *CharmLogger) Error(message string, keyvals ...interface{}) {
	l.internal.Error(message, keyvals...)
}

func (l *CharmLogger) Warn(message string, keyvals ...interface{}) {
	l.internal.Warn(message, keyvals...)
}

func (l *CharmLogger) Trace(message string, keyvals ...interface{}) {
	l.internal.Debug(message, keyvals...)
}

func LoggerAsHCLogger(l Logger) hclog.Logger {
	lo := hclog.LoggerOptions{}
	lo.Level = hclog.LevelFromString(l.Level())
	lo.Output = l.Output()

	return hclog.New(&lo)
}
