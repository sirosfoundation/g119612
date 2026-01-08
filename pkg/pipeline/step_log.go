package pipeline

import (
	"strings"

	"github.com/sirosfoundation/g119612/pkg/logging"
)

// Echo is a pipeline step that does nothing and simply returns the context unchanged.
// This is useful for testing pipeline configuration or as a placeholder step.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: Optional arguments that are ignored
//
// Returns:
//   - *Context: The same context that was passed in, unmodified
//   - error: Always nil
//
// Example usage in pipeline configuration:
//   - echo  # No-op step for testing
//   - echo:any_argument  # Arguments are allowed but ignored
func Echo(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	return ctx, nil
}

// Log is a pipeline step that outputs a log message to the console.
// This is useful for adding debug information or progress updates in the pipeline.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: String slice with args[0] being the message to log
//
// Example usage in pipeline YAML:
//
//   - log:
//   - "Processing complete: 10 TSLs transformed to HTML"
//
// Log outputs a message using the pipeline's logger.
// This function supports structured logging with fields in key=value format.
// The first argument is the log message, subsequent arguments are treated as fields.
//
// Example usage in YAML:
//
//   - log:
//   - Processing complete
//   - count=5
//   - filename=tsl.xml
//
// Level can be specified with a "level=" prefix in the message:
//
//   - log:
//   - level=debug Debug information
//   - key=value
//
// Parameters:
//   - pl: The pipeline containing the logger
//   - ctx: The current context
//   - args: First arg is message, subsequent args are fields in key=value format
//
// Returns:
//   - The unmodified context
//   - No error in normal operation
func Log(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) == 0 {
		return ctx, nil
	}

	// Pipeline always has a logger
	logger := pl.Logger

	// Parse the message and check for level prefix
	message := args[0]
	level := logging.InfoLevel // default level

	if strings.HasPrefix(message, "level=") {
		parts := strings.SplitN(message, " ", 2)
		levelStr := strings.TrimPrefix(parts[0], "level=")

		switch strings.ToLower(levelStr) {
		case "debug":
			level = logging.DebugLevel
		case "info":
			level = logging.InfoLevel
		case "warn", "warning":
			level = logging.WarnLevel
		case "error":
			level = logging.ErrorLevel
		case "fatal":
			level = logging.FatalLevel
		}

		if len(parts) > 1 {
			message = parts[1]
		} else {
			message = ""
		}
	}

	// Parse additional fields
	var fields []logging.Field
	for i := 1; i < len(args); i++ {
		arg := args[i]
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			fields = append(fields, logging.F(parts[0], parts[1]))
		}
	}

	// Log with the appropriate level
	switch level {
	case logging.DebugLevel:
		logger.Debug(message, fields...)
	case logging.InfoLevel:
		logger.Info(message, fields...)
	case logging.WarnLevel:
		logger.Warn(message, fields...)
	case logging.ErrorLevel:
		logger.Error(message, fields...)
	case logging.FatalLevel:
		logger.Fatal(message, fields...)
	}

	return ctx, nil
}
