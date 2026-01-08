// Package pipeline provides a framework for processing Trust Status Lists (TSLs)
// using a sequence of configurable steps defined in YAML.
package pipeline

import (
	"fmt"
	"os"

	"github.com/sirosfoundation/g119612/pkg/logging"
	"gopkg.in/yaml.v3"
)

// Pipeline represents a sequence of processing steps (Pipes) to be executed in order.
// Each Pipe calls a registered function with specified arguments to process Trust Status Lists.
// The Pipeline is typically loaded from a YAML file that defines a list of steps.
//
// The Pipeline always has a Logger available for use by pipeline steps.
// If no logger is specified during initialization, a default logger is used.
//
// Note: Configuration is NOT stored in the pipeline YAML. All configuration should
// be provided via command line arguments. Pipeline YAML files should contain only steps.
type Pipeline struct {
	Pipes  []Pipe         // The ordered list of pipeline steps to execute
	Logger logging.Logger // Logger for pipeline operations (never nil)
}

// Process executes all the steps in the pipeline in sequence, passing the Context from one step to the next.
// Each step modifies the Context and returns either a modified Context or an error.
// If a step returns an error, pipeline processing stops and the error is returned.
//
// Parameters:
//   - ctx: The initial Context to pass to the first step of the pipeline
//
// Returns:
//   - A pointer to the final Context after all steps have been executed
//   - An error if any step fails
func (pl *Pipeline) Process(ctx *Context) (*Context, error) {
	for i, pipe := range pl.Pipes {
		fn, ok := GetFunctionByName(pipe.MethodName)
		if !ok {
			return nil, fmt.Errorf("step %d: unknown methodName '%s'", i, pipe.MethodName)
		}
		var err error
		ctx, err = fn(pl, ctx, pipe.MethodArguments...)
		if err != nil {
			return ctx, fmt.Errorf("step %d (%s) failed: %w", i, pipe.MethodName, err)
		}
	}
	return ctx, nil
}

// NewPipeline loads a pipeline from a YAML file and returns a new Pipeline instance.
// The YAML file must contain a sequence of steps, where each step is a map with a single key
// (the method name) and a list of string arguments.
//
// IMPORTANT: The pipeline YAML should only contain steps, not configuration.
// All configuration should be provided via command-line arguments.
//
// Example YAML format:
//
//	# Pipeline steps as a direct list (no 'steps:' or 'config:' keys)
//	- load:
//		- https://example.com/tsl.xml
//	- transform:
//		- /path/to/stylesheet.xslt
//	- publish:
//		- /path/to/output
//
// Parameters:
//   - filename: Path to the YAML pipeline file
//
// Returns:
//   - A new Pipeline instance with the steps loaded from the YAML file
//   - An error if the file cannot be opened or parsed
func NewPipeline(filename string) (*Pipeline, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Always use the default logger - configuration should come from cmdline args, not pipeline files
	logger := logging.DefaultLogger()

	// Parse the pipeline as a simple list of pipes (no config sections)
	var pipes []Pipe
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&pipes); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline YAML: %w", err)
	}

	// Create a new pipeline with the parsed pipes
	return &Pipeline{
		Pipes:  pipes,
		Logger: logger,
	}, nil
}

// Pipe represents a single step in the pipeline with its method name and arguments.
// It provides custom YAML unmarshalling to parse the pipeline configuration format.
// Each Pipe corresponds to a registered StepFunc that will be executed during pipeline processing.
type Pipe struct {
	MethodName      string   // The name of the registered function to call
	MethodArguments []string // The arguments to pass to the function
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for custom YAML parsing.
// It expects a mapping node with exactly one key (the method name) and one value (a sequence of arguments).
//
// Example YAML structure:
//
//   - methodName:
//   - arg1
//   - arg2
//   - arg3
//
// Parameters:
//   - value: The YAML node to unmarshal
//
// Returns:
//   - An error if the YAML structure doesn't match the expected format
func (p *Pipe) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode || len(value.Content) != 2 {
		return &yaml.TypeError{Errors: []string{"Pipe must be a map with a single key (method name) and a list of arguments"}}
	}
	methodNode := value.Content[0]
	argsNode := value.Content[1]
	p.MethodName = methodNode.Value
	if argsNode.Kind != yaml.SequenceNode {
		return &yaml.TypeError{Errors: []string{"Pipe arguments must be a sequence"}}
	}
	p.MethodArguments = make([]string, len(argsNode.Content))
	for i, arg := range argsNode.Content {
		p.MethodArguments[i] = arg.Value
	}
	return nil
}

// WithLogger returns a new Pipeline with the specified logger.
// This allows for easy reconfiguration of the logger while preserving
// the rest of the pipeline steps.
//
// Parameters:
//   - logger: The new logger to use for the pipeline
//
// Returns:
//   - A new Pipeline instance with the same steps but using the specified logger
func (pl *Pipeline) WithLogger(logger logging.Logger) *Pipeline {
	if logger == nil {
		logger = logging.DefaultLogger()
	}
	return &Pipeline{
		Pipes:  pl.Pipes,
		Logger: logger,
	}
}
