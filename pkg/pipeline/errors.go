// Package pipeline provides error types for pipeline operations.
package pipeline

import (
	"errors"
	"fmt"
)

// Common sentinel errors for pipeline operations.
var (
	// ErrNoTSLs indicates that no TSLs are available in the context.
	ErrNoTSLs = errors.New("no TSLs available in context")

	// ErrInvalidArguments indicates that invalid arguments were provided to a pipeline step.
	ErrInvalidArguments = errors.New("invalid pipeline step arguments")

	// ErrEmptyPipeline indicates that the pipeline has no steps to execute.
	ErrEmptyPipeline = errors.New("pipeline has no steps to execute")

	// ErrFunctionNotFound indicates that a pipeline function was not found in the registry.
	ErrFunctionNotFound = errors.New("pipeline function not found")
)

// TSLLoadError represents an error that occurred while loading a TSL.
type TSLLoadError struct {
	URL    string // The URL or path that failed to load
	Reason string // Human-readable reason for the failure
	Err    error  // The underlying error
}

func (e *TSLLoadError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("failed to load TSL from %s: %s: %v", e.URL, e.Reason, e.Err)
	}
	return fmt.Sprintf("failed to load TSL from %s: %v", e.URL, e.Err)
}

func (e *TSLLoadError) Unwrap() error {
	return e.Err
}

// NewTSLLoadError creates a new TSLLoadError.
func NewTSLLoadError(url string, err error) *TSLLoadError {
	return &TSLLoadError{
		URL: url,
		Err: err,
	}
}

// NewTSLLoadErrorWithReason creates a new TSLLoadError with a specific reason.
func NewTSLLoadErrorWithReason(url, reason string, err error) *TSLLoadError {
	return &TSLLoadError{
		URL:    url,
		Reason: reason,
		Err:    err,
	}
}

// XSLTTransformError represents an error that occurred during XSLT transformation.
type XSLTTransformError struct {
	StylesheetPath string // Path to the XSLT stylesheet
	TSLIndex       int    // Index of the TSL being transformed
	Err            error  // The underlying error
}

func (e *XSLTTransformError) Error() string {
	return fmt.Sprintf("XSLT transformation failed for TSL %d using stylesheet %s: %v",
		e.TSLIndex, e.StylesheetPath, e.Err)
}

func (e *XSLTTransformError) Unwrap() error {
	return e.Err
}

// NewXSLTTransformError creates a new XSLTTransformError.
func NewXSLTTransformError(stylesheetPath string, tslIndex int, err error) *XSLTTransformError {
	return &XSLTTransformError{
		StylesheetPath: stylesheetPath,
		TSLIndex:       tslIndex,
		Err:            err,
	}
}

// ValidationError represents a validation error in pipeline processing.
type ValidationError struct {
	Field   string // The field that failed validation
	Value   string // The invalid value
	Message string // Human-readable error message
}

func (e *ValidationError) Error() string {
	if e.Field != "" && e.Value != "" {
		return fmt.Sprintf("validation error for %s='%s': %s", e.Field, e.Value, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// PublishError represents an error that occurred while publishing TSLs.
type PublishError struct {
	OutputPath string // The output path where publishing failed
	TSLCount   int    // Number of TSLs attempted to publish
	Err        error  // The underlying error
}

func (e *PublishError) Error() string {
	return fmt.Sprintf("failed to publish %d TSL(s) to %s: %v", e.TSLCount, e.OutputPath, e.Err)
}

func (e *PublishError) Unwrap() error {
	return e.Err
}

// NewPublishError creates a new PublishError.
func NewPublishError(outputPath string, tslCount int, err error) *PublishError {
	return &PublishError{
		OutputPath: outputPath,
		TSLCount:   tslCount,
		Err:        err,
	}
}

// CertificateError represents an error related to certificate processing.
type CertificateError struct {
	Operation string // The operation that failed (e.g., "parse", "validate")
	Subject   string // Certificate subject or identifier
	Err       error  // The underlying error
}

func (e *CertificateError) Error() string {
	if e.Subject != "" {
		return fmt.Sprintf("certificate %s failed for %s: %v", e.Operation, e.Subject, e.Err)
	}
	return fmt.Sprintf("certificate %s failed: %v", e.Operation, e.Err)
}

func (e *CertificateError) Unwrap() error {
	return e.Err
}

// NewCertificateError creates a new CertificateError.
func NewCertificateError(operation, subject string, err error) *CertificateError {
	return &CertificateError{
		Operation: operation,
		Subject:   subject,
		Err:       err,
	}
}

// PipelineStepError represents an error that occurred in a specific pipeline step.
type PipelineStepError struct {
	StepName  string   // Name of the pipeline step
	StepIndex int      // Index of the step in the pipeline
	Args      []string // Arguments passed to the step
	Err       error    // The underlying error
}

func (e *PipelineStepError) Error() string {
	return fmt.Sprintf("step %d (%s) failed: %v", e.StepIndex, e.StepName, e.Err)
}

func (e *PipelineStepError) Unwrap() error {
	return e.Err
}

// NewPipelineStepError creates a new PipelineStepError.
func NewPipelineStepError(stepName string, stepIndex int, args []string, err error) *PipelineStepError {
	return &PipelineStepError{
		StepName:  stepName,
		StepIndex: stepIndex,
		Args:      args,
		Err:       err,
	}
}
