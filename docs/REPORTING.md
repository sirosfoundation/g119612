# Pipeline Reporting Feature

The g119612 pipeline includes a reporting feature that generates comprehensive HTML reports documenting pipeline execution, certificate parsing statistics, and any issues encountered during processing.

## Overview

Reports are **generated automatically** by the `generate_index` step. When you run a pipeline with `generate_index`, both `index.html` and `report.html` are created in the same directory.

The HTML report includes:

- **Summary cards**: Error, warning, info counts, certificates parsed/skipped
- **Pipeline steps list**: Numbered execution order of pipeline steps
- **Issues table**: Searchable and filterable by severity and step name
- **Certificate parsing details**: Per-TSL expandable sections with progress bars
- **Interactive features**: Search/filter controls

## Usage

### Default (Report Included)

```yaml
- generate_index:
    - /var/www/html/tsl
    - "EU Trust Lists"
```

This generates both `index.html` and `report.html` in `/var/www/html/tsl/`.

### Skip Report Generation

```yaml
- generate_index:
    - /var/www/html/tsl
    - "EU Trust Lists"
    - no-report
```

### Standalone Report (Deprecated)

The separate `report` step is still available for backward compatibility or LoTE-only pipelines:

```yaml
- report:
    - /output/report.html
    - "Pipeline Report"
```

## Data Structures

### PipelineReport

The main container for all report data:

| Field       | Type            | Description                              |
|-------------|-----------------|------------------------------------------|
| `Title`     | string          | Report title                             |
| `Issues`    | []ReportIssue   | All issues in order of recording         |
| `CertStats` | []CertSummary   | Per-source certificate parsing stats     |
| `StepsRun`  | []string        | Names of pipeline steps executed         |
| `StartTime` | time.Time       | Pipeline start time                      |

### ReportIssue

A single finding recorded during execution:

| Field       | Type     | Description                                    |
|-------------|----------|------------------------------------------------|
| `Severity`  | Severity | `info`, `warning`, or `error`                  |
| `Step`      | string   | Pipeline step name (e.g. "load", "select")     |
| `Source`    | string   | TSL source URL or file path                    |
| `TSP`       | string   | Trust Service Provider name (when applicable)  |
| `Service`   | string   | Trust service name (when applicable)           |
| `Message`   | string   | Human-readable description                     |
| `Detail`    | string   | Additional detail (certificate subject, etc.)  |
| `Timestamp` | time.Time| When the issue was recorded                    |

### CertSummary

Aggregated certificate parsing statistics per TSL source:

| Field     | Type                      | Description                        |
|-----------|---------------------------|------------------------------------|
| `Source`  | string                    | TSL source URL or file             |
| `Stats`   | *etsi119612.CertParseStats| Aggregate stats for this source    |
| `Details` | []CertIssueDetail         | Individual certificate issues      |

## Automatic Issue Collection

The report is automatically populated during pipeline execution:

1. **Step Tracking**: Each step name is recorded via `RecordStep()`
2. **Error Capture**: Failed steps automatically add error-level issues
3. **Certificate Stats**: The `select` step records parsing statistics per TSL

### Programmatic Access

Any pipeline step can add issues to the report:

```go
// Ensure report exists on context
ctx.EnsureReport()

// Add a simple issue
ctx.Report.AddIssue(SeverityWarning, "mystep", sourceURL, "message")

// Add a detailed issue
ctx.Report.AddDetailedIssue(ReportIssue{
    Severity: SeverityError,
    Step:     "validate",
    Source:   "https://example.com/tsl.xml",
    TSP:      "Example TSP",
    Service:  "CA Service",
    Message:  "Certificate expired",
    Detail:   "CN=Example CA, expires 2024-01-15",
})

// Add certificate summary
ctx.Report.AddCertSummary(source, stats, details)
```

## Report Metrics

The report provides these aggregate metrics:

| Method                | Returns          | Description                           |
|-----------------------|------------------|---------------------------------------|
| `CountBySeverity()`   | map[Severity]int | Issue counts per severity level       |
| `TotalCertsParsed()`  | int              | Total successfully parsed certs       |
| `TotalCertsSkipped()` | int              | Total certs that could not be parsed  |
| `HasIssues()`         | bool             | Whether any issues were recorded      |

## HTML Report Features

The generated HTML report includes:

- **Responsive design** using PicoCSS
- **Search** across all issue text
- **Filter dropdowns** for severity and step name
- **Progress bars** for certificate parsing rates
- **Expandable sections** for per-TSL certificate details
- **Table of skipped certs** grouped by error kind

---

## Complete Example: EU List of Lists

This example demonstrates processing the EU List of Lists (LOTL) with full reporting.

### Pipeline Configuration

Create `eu-lotl-pipeline.yaml`:

```yaml
# EU List of Lists Processing Pipeline
# =====================================
# Fetches the EU LOTL, traverses all member state TSLs,
# generates HTML versions, and produces index + report.

# Configure HTTP client for fetching TSLs
- set-fetch-options:
    - user-agent:g119612-TSL-Tool/1.0
    - timeout:120s

# Load the EU List of Lists (root of EU trust infrastructure)
- load:
    - https://ec.europa.eu/tools/lotl/eu-lotl.xml

# Traverse to member state TSLs (depth 2: LOTL -> pointers -> member TSLs)
# This builds a certificate pool from all referenced lists
- select:
    - reference-depth:2

# Transform each TSL to HTML using the embedded XSLT stylesheet
- transform:
    - embedded:tsl-to-html.xslt
    - /var/www/html/tsl
    - html

# Generate index page AND report (report.html is created automatically)
- generate_index:
    - /var/www/html/tsl
    - "EU Trust Status Lists"
```

### Running the Pipeline

```bash
# Build the tool
make build

# Run the pipeline
./tsl-tool --log-level info eu-lotl-pipeline.yaml

# For verbose output including certificate parsing details:
./tsl-tool --log-level debug eu-lotl-pipeline.yaml
```

### Expected Output

The pipeline will:

1. **Fetch** the EU LOTL from `ec.europa.eu`
2. **Traverse** ~30+ member state TSLs referenced by the LOTL
3. **Parse certificates** from each TSL (typically 1000+ certs total)
4. **Generate HTML** files for each TSL
5. **Create index.html** — listing all processed TSLs with report summary banner
6. **Create report.html** — detailed pipeline execution report

### Sample Report Output

The report will show:

```
┌──────────────────────────────────────────────────────────────────┐
│              EU Trust Status Lists - Pipeline Report             │
│           Generated 2026-03-25 10:15:32 — ran in 45.2s          │
├────────────┬────────────┬──────────┬─────────────┬──────────────┤
│  0 Errors  │ 12 Warnings│  3 Info  │ 1,247 Parsed│  23 Skipped  │
└────────────┴────────────┴──────────┴─────────────┴──────────────┘

Pipeline Steps Executed (4):
  1. set-fetch-options  2. load  3. select  4. transform  5. generate_index

Issues (15):
┌──────────┬─────────┬─────────────────────────────────┬──────────────────────────┐
│ Severity │  Step   │ Source                          │ Message                  │
├──────────┼─────────┼─────────────────────────────────┼──────────────────────────┤
│ WARNING  │ select  │ https://tl.hungary.hu/tsl.xml   │ Certificate parse failed │
│ WARNING  │ select  │ https://tsl.belgium.be/tsl.xml  │ Unsupported key algorithm│
│ INFO     │ load    │ https://ec.europa.eu/.../lotl   │ Loaded 31 TSL pointers   │
└──────────┴─────────┴─────────────────────────────────┴──────────────────────────┘

Certificate Parsing Summary:

▼ https://ec.europa.eu/tools/lotl/eu-lotl.xml — 45/45 parsed
  ████████████████████████████████████████ 100%

▼ https://tl.germany.de/tsl.xml — 127/130 parsed (3 skipped)
  █████████████████████████████████████░░░ 98%
  ┌────────────────────┬───────┐
  │ Error Kind         │ Count │
  ├────────────────────┼───────┤
  │ unsupported_algo   │     2 │
  │ malformed_cert     │     1 │
  └────────────────────┴───────┘
```

### Extended Pipeline with LoTE Conversion

To also convert TSLs to the modern LoTE format:

```yaml
# EU List of Lists → LoTE Conversion
- set-fetch-options:
    - user-agent:g119612-TSL-Tool/1.0
    - timeout:120s

- load:
    - https://ec.europa.eu/tools/lotl/eu-lotl.xml

- select:
    - reference-depth:2

# Convert all TSLs to LoTE format
- convert-to-lote:

# Merge into a single LoTE
- merge-lote:

# Increment sequence number
- increment-lote-sequence:

# Publish LoTE JSON (with optional JWS signing)
- publish-lote:
    - /var/www/html/lote

# Also generate HTML versions with index + report
- transform:
    - embedded:tsl-to-html.xslt
    - /var/www/html/tsl
    - html

- generate_index:
    - /var/www/html/tsl
    - "EU Trust Status Lists"
```

For LoTE-only pipelines without HTML output, use the standalone `report` step:

```yaml
- load-lote:
    - https://example.org/lote.json

# ... processing ...

- report:
    - /var/www/html/lote-report.html
    - "LoTE Pipeline Report"
```

## Best Practices

1. **Use `generate_index`** — it generates reports automatically, no separate step needed
2. **Use `no-report` sparingly** — most pipelines benefit from having a report
3. **Review warnings** — certificate parsing warnings may indicate TSL quality issues
4. **Monitor over time** — compare reports to track changes in EU trust infrastructure
5. **Enable debug logging** — use `--log-level debug` for detailed certificate parsing info

## Troubleshooting

### Common Warning Types

| Warning                  | Cause                                          | Resolution                     |
|--------------------------|------------------------------------------------|--------------------------------|
| `unsupported_algo`       | Brainpool or other non-standard curve          | Ensure `go-cryptoutil` is enabled |
| `malformed_cert`         | Invalid certificate encoding in TSL            | Report to TSL operator          |
| `expired_cert`           | Certificate past validity period               | Informational, expected         |
| `fetch_failed`           | Network or HTTP error fetching TSL             | Check connectivity/URL          |

### Enabling Brainpool Support

For German (gematik) and other TSLs using brainpool curves:

```go
import (
    "github.com/sirosfoundation/go-cryptoutil"
    "github.com/sirosfoundation/go-cryptoutil/brainpool"
)

ext := cryptoutil.New()
brainpool.Register(ext)
ctx.CryptoExt = ext
```

The `tsl-tool` CLI registers this automatically.
