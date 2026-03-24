package pipeline

import (
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

//go:embed templates/report.html
var reportHTMLTemplate string

// certStatsView is a template-friendly projection of CertSummary.
type certStatsView struct {
	Source        string
	Stats         certStatsNumbers
	SkippedByKind map[etsi119612.CertParseErrorKind]int
	Details       []CertIssueDetail
}

// certStatsNumbers exposes CertParseStats fields directly for the template.
type certStatsNumbers struct {
	Total        int
	Parsed       int
	TotalSkipped int
}

// GenerateReport renders the accumulated pipeline report to an HTML file.
//
// Arguments:
//   - arg[0]: Output file path (e.g. "/var/www/report.html")
//   - arg[1]: (Optional) Report title (default: "Pipeline Report")
//
// Example usage in pipeline YAML:
//
//	- report: [/output/report.html, "EU LOTL Pipeline Report"]
func GenerateReport(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return ctx, fmt.Errorf("report: %w: missing output file path", ErrInvalidArguments)
	}
	outPath := args[0]
	title := "Pipeline Report"
	if len(args) >= 2 {
		title = args[1]
	}

	ctx.EnsureReport()
	report := ctx.Report
	report.Title = title

	// Ensure the output directory exists.
	if dir := filepath.Dir(outPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return ctx, fmt.Errorf("report: cannot create output directory: %w", err)
		}
	}

	counts := report.CountBySeverity()

	// Build template-friendly cert stats.
	var certViews []certStatsView
	for _, cs := range report.CertStats {
		v := certStatsView{
			Source:  cs.Source,
			Details: cs.Details,
		}
		if cs.Stats != nil {
			v.Stats = certStatsNumbers{
				Total:        cs.Stats.Total,
				Parsed:       cs.Stats.Parsed,
				TotalSkipped: cs.Stats.TotalSkipped(),
			}
			v.SkippedByKind = cs.Stats.Skipped
		}
		certViews = append(certViews, v)
	}

	// Collect unique step names for the filter dropdown.
	stepSet := map[string]struct{}{}
	for _, issue := range report.Issues {
		if issue.Step != "" {
			stepSet[issue.Step] = struct{}{}
		}
	}
	var uniqueSteps []string
	for s := range stepSet {
		uniqueSteps = append(uniqueSteps, s)
	}

	duration := ""
	if !report.StartTime.IsZero() {
		duration = time.Since(report.StartTime).Round(time.Millisecond).String()
	}

	data := struct {
		Title         string
		GeneratedDate string
		Duration      string
		ErrorCount    int
		WarningCount  int
		InfoCount     int
		CertsParsed   int
		CertsSkipped  int
		Steps         []string
		Issues        []ReportIssue
		CertStats     []certStatsView
		UniqueSteps   []string
	}{
		Title:         title,
		GeneratedDate: time.Now().Format("2006-01-02 15:04:05"),
		Duration:      duration,
		ErrorCount:    counts[SeverityError],
		WarningCount:  counts[SeverityWarning],
		InfoCount:     counts[SeverityInfo],
		CertsParsed:   report.TotalCertsParsed(),
		CertsSkipped:  report.TotalCertsSkipped(),
		Steps:         report.StepsRun,
		Issues:        report.Issues,
		CertStats:     certViews,
		UniqueSteps:   uniqueSteps,
	}

	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
		"percent": func(part, total int) int {
			if total == 0 {
				return 0
			}
			return part * 100 / total
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(reportHTMLTemplate)
	if err != nil {
		return ctx, fmt.Errorf("report: template parse error: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return ctx, fmt.Errorf("report: cannot create output file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return ctx, fmt.Errorf("report: template execution error: %w", err)
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Pipeline report generated",
			logging.F("path", outPath),
			logging.F("errors", counts[SeverityError]),
			logging.F("warnings", counts[SeverityWarning]),
			logging.F("info", counts[SeverityInfo]),
			logging.F("cert_sources", len(report.CertStats)))
	}

	return ctx, nil
}

func init() {
	RegisterFunction("report", GenerateReport)
}
