package logging

import (
	"bytes"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSilentLogger(t *testing.T) {
	logger := SilentLogger()
	require.NotNil(t, logger)

	// Ensure logging does not panic
	logger.Debug("should not appear")
	logger.Info("should not appear")
	logger.Warn("should not appear")
	logger.Error("should not appear")
}

func TestSetOutput(t *testing.T) {
	var buf bytes.Buffer

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(io.Discard)
	logrusLogger.SetLevel(logrus.DebugLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})

	adapter := NewLogrusAdapter(logrusLogger)

	// Redirect to buffer
	adapter.SetOutput(&buf)
	adapter.Info("hello setoutput")

	assert.Contains(t, buf.String(), "hello setoutput")
}

func TestSetOutput_NonWriter(t *testing.T) {
	logrusLogger := logrus.New()
	adapter := NewLogrusAdapter(logrusLogger)

	// Passing a non-io.Writer should be silently ignored
	adapter.SetOutput("not-a-writer")
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.DebugLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})

	adapter := NewLogrusAdapter(logrusLogger)
	child := adapter.WithField("component", "test-comp")
	require.NotNil(t, child)

	child.Info("with-field-msg")
	assert.Contains(t, buf.String(), "component=test-comp")
	assert.Contains(t, buf.String(), "with-field-msg")
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetLevel(logrus.DebugLevel)
	logrusLogger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})

	adapter := NewLogrusAdapter(logrusLogger)
	child := adapter.WithFields(
		F("alpha", "one"),
		F("beta", 2),
	)
	require.NotNil(t, child)

	child.Info("with-fields-msg")
	assert.Contains(t, buf.String(), "alpha=one")
	assert.Contains(t, buf.String(), "beta=2")
	assert.Contains(t, buf.String(), "with-fields-msg")
}

func TestNewLogrusAdapter_NilLogger(t *testing.T) {
	// Passing nil logger should use standard logrus logger
	adapter := NewLogrusAdapter(nil)
	assert.NotNil(t, adapter)
}
