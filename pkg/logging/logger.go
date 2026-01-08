// Package logging provides a unified logging interface for the g119612 package.
package logging

import (
	"context"
)

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// DebugLevel logs are typically voluminous and are usually disabled in production.
	DebugLevel LogLevel = iota
	// InfoLevel logs are the default log level and represent normal application behavior.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel
	// ErrorLevel logs are high-priority and represent a failure in the application.
	ErrorLevel
	// FatalLevel logs are very severe errors that typically cause the application to terminate.
	FatalLevel
)

// Logger is the interface that provides structured logging methods.
type Logger interface {
	// Debug logs a message with debug level.
	Debug(msg string, fields ...Field)
	// Info logs a message with info level.
	Info(msg string, fields ...Field)
	// Warn logs a message with warn level.
	Warn(msg string, fields ...Field)
	// Error logs a message with error level.
	Error(msg string, fields ...Field)
	// Fatal logs a message with fatal level and then exits with status code 1.
	Fatal(msg string, fields ...Field)

	// WithContext returns a logger with the given context.
	WithContext(ctx context.Context) Logger
	// WithField adds a field to the logger and returns a new logger.
	WithField(key string, value interface{}) Logger
	// WithFields adds multiple fields to the logger and returns a new logger.
	WithFields(fields ...Field) Logger

	// GetLevel returns the current logging level.
	GetLevel() LogLevel
	// SetLevel sets the logging level.
	SetLevel(level LogLevel)
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new Field with the given key and value.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// OutputConfigurable defines an interface for loggers that can have their output configured.
type OutputConfigurable interface {
	// SetOutput sets the output for the logger.
	SetOutput(out interface{})
}
