package pipeline

import (
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// Severity classifies the importance of a report issue.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// ReportIssue represents a single finding recorded during pipeline execution.
type ReportIssue struct {
	Severity  Severity  // info, warning, or error
	Step      string    // Pipeline step that produced this issue (e.g. "load", "select")
	Source    string    // TSL source URL or file path, when applicable
	TSP       string    // Trust Service Provider name, when applicable
	Service   string    // Trust service name, when applicable
	Message   string    // Human-readable description of the issue
	Detail    string    // Optional additional detail (e.g. certificate subject, error text)
	Timestamp time.Time // When the issue was recorded
}

// CertSummary aggregates certificate parsing statistics per TSL source.
type CertSummary struct {
	Source  string                     // TSL source URL or file path
	Stats   *etsi119612.CertParseStats // Aggregate stats for this source
	Details []CertIssueDetail          // Individual certificate issues
}

// CertIssueDetail describes a single certificate parsing failure.
type CertIssueDetail struct {
	TSP       string                        // Trust Service Provider name
	Service   string                        // Trust service name
	ErrorKind etsi119612.CertParseErrorKind // Classification of the error
	Detail    string                        // Error message or certificate subject hint
}

// PipelineReport accumulates issues and statistics across all pipeline steps.
// It is attached to the pipeline Context and can be populated by any step.
// The "report" pipeline step renders its contents to an HTML file.
type PipelineReport struct {
	Title     string        // Report title (set by the report step or caller)
	Issues    []ReportIssue // All issues, in order of recording
	CertStats []CertSummary // Per-source certificate parsing summaries
	StepsRun  []string      // Names of steps executed so far
	StartTime time.Time     // When the pipeline started
}

// NewPipelineReport creates an empty report, recording the current time as start.
func NewPipelineReport() *PipelineReport {
	return &PipelineReport{
		StartTime: time.Now(),
	}
}

// AddIssue records a single issue into the report.
func (r *PipelineReport) AddIssue(sev Severity, step, source, message string) {
	r.Issues = append(r.Issues, ReportIssue{
		Severity:  sev,
		Step:      step,
		Source:    source,
		Message:   message,
		Timestamp: time.Now(),
	})
}

// AddDetailedIssue records an issue with full context.
func (r *PipelineReport) AddDetailedIssue(issue ReportIssue) {
	if issue.Timestamp.IsZero() {
		issue.Timestamp = time.Now()
	}
	r.Issues = append(r.Issues, issue)
}

// AddCertSummary records certificate parsing statistics for a TSL source.
func (r *PipelineReport) AddCertSummary(source string, stats *etsi119612.CertParseStats, details []CertIssueDetail) {
	r.CertStats = append(r.CertStats, CertSummary{
		Source:  source,
		Stats:   stats,
		Details: details,
	})
}

// RecordStep logs that a pipeline step was executed.
func (r *PipelineReport) RecordStep(name string) {
	r.StepsRun = append(r.StepsRun, name)
}

// CountBySeverity returns the number of issues at each severity level.
func (r *PipelineReport) CountBySeverity() map[Severity]int {
	counts := map[Severity]int{
		SeverityInfo:    0,
		SeverityWarning: 0,
		SeverityError:   0,
	}
	for _, issue := range r.Issues {
		counts[issue.Severity]++
	}
	return counts
}

// TotalCertsParsed returns the aggregate number of successfully parsed certificates.
func (r *PipelineReport) TotalCertsParsed() int {
	n := 0
	for _, cs := range r.CertStats {
		if cs.Stats != nil {
			n += cs.Stats.Parsed
		}
	}
	return n
}

// TotalCertsSkipped returns the aggregate number of certificates that could not be parsed.
func (r *PipelineReport) TotalCertsSkipped() int {
	n := 0
	for _, cs := range r.CertStats {
		if cs.Stats != nil {
			n += cs.Stats.TotalSkipped()
		}
	}
	return n
}

// HasIssues returns true if any issues have been recorded.
func (r *PipelineReport) HasIssues() bool {
	return len(r.Issues) > 0
}
