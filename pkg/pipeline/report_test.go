package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPipelineReport(t *testing.T) {
	r := NewPipelineReport()
	require.NotNil(t, r)
	assert.False(t, r.StartTime.IsZero())
	assert.Empty(t, r.Issues)
	assert.Empty(t, r.CertStats)
	assert.Empty(t, r.StepsRun)
}

func TestReportAddIssue(t *testing.T) {
	r := NewPipelineReport()
	r.AddIssue(SeverityWarning, "select", "https://example.org/tsl.xml", "3 certs skipped")
	require.Len(t, r.Issues, 1)
	assert.Equal(t, SeverityWarning, r.Issues[0].Severity)
	assert.Equal(t, "select", r.Issues[0].Step)
	assert.Equal(t, "https://example.org/tsl.xml", r.Issues[0].Source)
	assert.False(t, r.Issues[0].Timestamp.IsZero())
}

func TestReportAddDetailedIssue(t *testing.T) {
	r := NewPipelineReport()
	r.AddDetailedIssue(ReportIssue{
		Severity: SeverityError,
		Step:     "load",
		Source:   "https://example.org/broken.xml",
		TSP:      "Acme CA",
		Service:  "Qualified CA",
		Message:  "certificate parse failed",
		Detail:   "unsupported elliptic curve",
	})
	require.Len(t, r.Issues, 1)
	assert.Equal(t, "Acme CA", r.Issues[0].TSP)
	assert.False(t, r.Issues[0].Timestamp.IsZero(), "timestamp should be set automatically")
}

func TestReportCountBySeverity(t *testing.T) {
	r := NewPipelineReport()
	r.AddIssue(SeverityError, "load", "", "fail1")
	r.AddIssue(SeverityError, "load", "", "fail2")
	r.AddIssue(SeverityWarning, "select", "", "warn1")
	r.AddIssue(SeverityInfo, "load", "", "info1")

	counts := r.CountBySeverity()
	assert.Equal(t, 2, counts[SeverityError])
	assert.Equal(t, 1, counts[SeverityWarning])
	assert.Equal(t, 1, counts[SeverityInfo])
}

func TestReportCertTotals(t *testing.T) {
	r := NewPipelineReport()
	stats1 := etsi119612.NewCertParseStats()
	stats1.RecordSuccess()
	stats1.RecordSuccess()
	stats1.RecordSkip(etsi119612.CertParseErrUnsupportedCurve)

	stats2 := etsi119612.NewCertParseStats()
	stats2.RecordSuccess()

	r.AddCertSummary("source1", stats1, nil)
	r.AddCertSummary("source2", stats2, nil)

	assert.Equal(t, 3, r.TotalCertsParsed())
	assert.Equal(t, 1, r.TotalCertsSkipped())
}

func TestReportRecordStep(t *testing.T) {
	r := NewPipelineReport()
	r.RecordStep("load")
	r.RecordStep("select")
	r.RecordStep("report")
	assert.Equal(t, []string{"load", "select", "report"}, r.StepsRun)
}

func TestReportHasIssues(t *testing.T) {
	r := NewPipelineReport()
	assert.False(t, r.HasIssues())
	r.AddIssue(SeverityInfo, "load", "", "loaded")
	assert.True(t, r.HasIssues())
}

func TestContextEnsureReport(t *testing.T) {
	ctx := &Context{}
	assert.Nil(t, ctx.Report)
	ctx.EnsureReport()
	assert.NotNil(t, ctx.Report)
	// Second call should not replace it.
	first := ctx.Report
	ctx.EnsureReport()
	assert.Same(t, first, ctx.Report)
}

func TestPipelineProcessRecordsSteps(t *testing.T) {
	pl := &Pipeline{
		Pipes: []Pipe{
			{MethodName: "echo", MethodArguments: nil},
		},
		Logger: logging.DefaultLogger(),
	}
	ctx := &Context{}
	ctx.EnsureReport()
	out, err := pl.Process(ctx)
	require.NoError(t, err)
	assert.Contains(t, out.Report.StepsRun, "echo")
}

func TestGenerateReportStep(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")

	ctx := &Context{}
	ctx.EnsureReport()
	ctx.Report.AddIssue(SeverityWarning, "select", "https://example.org/tsl.xml", "2 certs skipped")
	ctx.Report.AddIssue(SeverityError, "load", "https://broken.example.org", "connection refused")
	ctx.Report.AddIssue(SeverityInfo, "load", "https://ok.example.org", "Loaded 42 TSLs")

	stats := etsi119612.NewCertParseStats()
	stats.RecordSuccess()
	stats.RecordSuccess()
	stats.RecordSkip(etsi119612.CertParseErrUnsupportedCurve)
	ctx.Report.AddCertSummary("https://example.org/tsl.xml", stats, []CertIssueDetail{
		{TSP: "Acme", Service: "CA/QC", ErrorKind: etsi119612.CertParseErrUnsupportedCurve},
	})
	ctx.Report.RecordStep("load")
	ctx.Report.RecordStep("select")

	pl := &Pipeline{Logger: logging.DefaultLogger()}

	out, err := GenerateReport(pl, ctx, outPath, "Test Report")
	require.NoError(t, err)
	assert.NotNil(t, out)

	// Verify the file was created and contains expected content.
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	html := string(data)
	assert.Contains(t, html, "Test Report")
	assert.Contains(t, html, "2 certs skipped")
	assert.Contains(t, html, "connection refused")
	assert.Contains(t, html, "unsupported_elliptic_curve")
	assert.Contains(t, html, "Acme")
	assert.Contains(t, html, "badge-error")
	assert.Contains(t, html, "badge-warning")
}

func TestGenerateReportStepEmptyReport(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "empty.html")

	ctx := &Context{}
	ctx.EnsureReport()

	pl := &Pipeline{Logger: logging.DefaultLogger()}
	_, err := GenerateReport(pl, ctx, outPath)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "No Issues Found")
}

func TestGenerateReportStepMissingArgs(t *testing.T) {
	ctx := &Context{}
	pl := &Pipeline{Logger: logging.DefaultLogger()}
	_, err := GenerateReport(pl, ctx)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "missing output file path"))
}

func TestGenerateReportCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "sub", "dir", "report.html")

	ctx := &Context{}
	ctx.EnsureReport()

	pl := &Pipeline{Logger: logging.DefaultLogger()}
	_, err := GenerateReport(pl, ctx, outPath)
	require.NoError(t, err)
	assert.FileExists(t, outPath)
}
