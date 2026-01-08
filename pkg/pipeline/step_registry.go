package pipeline

import "sync"

// StepFunc is the function type for pipeline steps.
// Each step takes a pipeline instance, a context, and variadic string arguments,
// processes the context according to its logic, and returns either a modified context or an error.
//
// Parameters:
//   - pl: The pipeline instance (provides access to the logger)
//   - ctx: The current context with TSLs and certificate pools
//   - args: String arguments from the pipeline step definition
//
// Returns:
//   - A modified Context after processing
//   - An error if processing fails
type StepFunc func(pl *Pipeline, ctx *Context, args ...string) (*Context, error)

var (
	functionRegistry = make(map[string]StepFunc)
	registryMutex    sync.RWMutex
)

// RegisterFunction registers a pipeline step function with the given name.
// Once registered, the function can be referenced by name in pipeline YAML files
// and will be looked up during pipeline processing.
//
// This function is thread-safe due to mutex protection.
//
// Parameters:
//   - name: A unique name to identify the step function in pipeline configurations
//   - fn: The StepFunc implementation to register
func RegisterFunction(name string, fn StepFunc) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	functionRegistry[name] = fn
}

// GetFunctionByName retrieves a registered pipeline step function by name.
// It returns the function and a boolean indicating whether it was found.
//
// This function is thread-safe due to mutex protection.
//
// Parameters:
//   - name: The name of the function to look up
//
// Returns:
//   - The registered StepFunc, if found
//   - A boolean indicating whether the function was found
func GetFunctionByName(name string) (StepFunc, bool) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	fn, ok := functionRegistry[name]
	return fn, ok
}
