package logging

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
)

// LogrusAdapter implements the Logger interface using logrus.
type LogrusAdapter struct {
	logger *logrus.Entry
}

// NewLogrusAdapter creates a new LogrusAdapter with the given logrus logger.
func NewLogrusAdapter(logger *logrus.Logger) *LogrusAdapter {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &LogrusAdapter{
		logger: logrus.NewEntry(logger),
	}
}

// Debug logs a message with debug level.
func (l *LogrusAdapter) Debug(msg string, fields ...Field) {
	l.logger.WithFields(convertFields(fields)).Debug(msg)
}

// Info logs a message with info level.
func (l *LogrusAdapter) Info(msg string, fields ...Field) {
	l.logger.WithFields(convertFields(fields)).Info(msg)
}

// Warn logs a message with warn level.
func (l *LogrusAdapter) Warn(msg string, fields ...Field) {
	l.logger.WithFields(convertFields(fields)).Warn(msg)
}

// Error logs a message with error level.
func (l *LogrusAdapter) Error(msg string, fields ...Field) {
	l.logger.WithFields(convertFields(fields)).Error(msg)
}

// Fatal logs a message with fatal level and then exits with status code 1.
func (l *LogrusAdapter) Fatal(msg string, fields ...Field) {
	l.logger.WithFields(convertFields(fields)).Fatal(msg)
}

// WithContext returns a logger with the given context.
func (l *LogrusAdapter) WithContext(ctx context.Context) Logger {
	return &LogrusAdapter{
		logger: l.logger.WithContext(ctx),
	}
}

// WithField adds a field to the logger and returns a new logger.
func (l *LogrusAdapter) WithField(key string, value interface{}) Logger {
	return &LogrusAdapter{
		logger: l.logger.WithField(key, value),
	}
}

// WithFields adds multiple fields to the logger and returns a new logger.
func (l *LogrusAdapter) WithFields(fields ...Field) Logger {
	return &LogrusAdapter{
		logger: l.logger.WithFields(convertFields(fields)),
	}
}

// GetLevel returns the current logging level.
func (l *LogrusAdapter) GetLevel() LogLevel {
	switch l.logger.Logger.GetLevel() {
	case logrus.DebugLevel:
		return DebugLevel
	case logrus.InfoLevel:
		return InfoLevel
	case logrus.WarnLevel:
		return WarnLevel
	case logrus.ErrorLevel:
		return ErrorLevel
	case logrus.FatalLevel:
		return FatalLevel
	default:
		return InfoLevel
	}
}

// SetLevel sets the logging level.
func (l *LogrusAdapter) SetLevel(level LogLevel) {
	switch level {
	case DebugLevel:
		l.logger.Logger.SetLevel(logrus.DebugLevel)
	case InfoLevel:
		l.logger.Logger.SetLevel(logrus.InfoLevel)
	case WarnLevel:
		l.logger.Logger.SetLevel(logrus.WarnLevel)
	case ErrorLevel:
		l.logger.Logger.SetLevel(logrus.ErrorLevel)
	case FatalLevel:
		l.logger.Logger.SetLevel(logrus.FatalLevel)
	default:
		l.logger.Logger.SetLevel(logrus.InfoLevel)
	}
}

// convertFields converts our Field type to logrus.Fields.
func convertFields(fields []Field) logrus.Fields {
	logrusFields := logrus.Fields{}
	for _, field := range fields {
		logrusFields[field.Key] = field.Value
	}
	return logrusFields
}

// SetOutput sets the output for the logger.
// It implements the OutputConfigurable interface.
// The parameter should be an io.Writer such as os.Stdout or a file.
func (l *LogrusAdapter) SetOutput(out interface{}) {
	if writer, ok := out.(io.Writer); ok {
		l.logger.Logger.SetOutput(writer)
	}
}
