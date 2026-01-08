// Package main provides the tsl-tool command-line application for ETSI TSL processing.
//
// tsl-tool is a batch processor for ETSI TS 119612 Trust Status Lists (TSLs).
// It processes TSLs using a YAML-defined pipeline with configurable steps
// for loading, transforming, selecting certificates, and publishing TSLs.
//
// This tool is designed to run as a batch process (e.g., via cron) to:
// - Download and process TSLs from remote sources
// - Apply XSLT transformations to generate HTML documentation
// - Extract certificate pools from TSLs
// - Generate and sign new TSLs
// - Publish processed TSLs to files or directories
//
// # Pipeline Overview
//
// The pipeline consists of a sequence of steps defined in a YAML file:
//
//   - load:
//   - https://example.com/tsl.xml
//   - transform:
//   - embedded:tsl-to-html.xslt
//   - /output/html
//   - html
//   - select:
//   - reference-depth:2
//   - publish:
//   - /output/xml
//
// # Available Pipeline Steps
//
//   - load: Load TSL from URL or file path
//   - select: Build certificate pool from loaded TSLs
//   - transform: Apply XSLT transformation
//   - publish: Write TSLs to files
//   - generate: Generate new TSL from metadata
//   - log: Output messages to log
//   - set-fetch-options: Configure HTTP options
//   - echo: No-op placeholder step
//
// # Usage
//
//	tsl-tool [options] <pipeline.yaml>
//
// Options:
//
//	--help           Show help message
//	--version        Show version information
//	--log-level      Logging level: debug, info, warn, error (default: info)
//	--log-format     Logging format: text or json (default: text)
//	--output         Write certificate pool PEM to file (optional)
//
// # Exit Codes
//
//	0  Success
//	1  General error (invalid arguments, pipeline failure)
//
// See: https://github.com/sirosfoundation/g119612 for more information
package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/pipeline"
)

// Version is set at build time using -ldflags
var Version = "dev"

// parseLogLevel converts a string log level to the corresponding LogLevel enum value.
func parseLogLevel(level string) logging.LogLevel {
	level = strings.ToLower(level)
	switch level {
	case "debug":
		return logging.DebugLevel
	case "info":
		return logging.InfoLevel
	case "warn", "warning":
		return logging.WarnLevel
	case "error":
		return logging.ErrorLevel
	case "fatal":
		return logging.FatalLevel
	default:
		fmt.Fprintf(os.Stderr, "Warning: unknown log level '%s', using 'info'\n", level)
		return logging.InfoLevel
	}
}

// usage prints the command-line usage information.
func usage() {
	prog := os.Args[0]
	fmt.Fprintf(os.Stderr, `
tsl-tool: ETSI Trust Status List (TSL) Pipeline Processor

Usage: %s [options] <pipeline.yaml>

A batch processing tool for ETSI TS 119612 Trust Status Lists.
Designed to run as a cron job for periodic TSL processing.

Options:
  --help           Show this help message and exit
  --version        Show version information and exit
  --log-level      Logging level: debug, info, warn, error (default: info)
  --log-format     Logging format: text or json (default: text)
  --output         Write extracted certificate pool PEM to file (optional)

Pipeline Steps:
  load             Load TSL from URL or file path
  select           Build certificate pool from TSLs
  transform        Apply XSLT transformation
  publish          Write TSLs to files
  generate         Generate new TSL from metadata
  generate_index   Generate HTML index of TSL files
  log              Output messages to log
  set-fetch-options Configure HTTP fetch options
  echo             No-op placeholder step

Example:
  %s --log-level debug pipeline.yaml
  %s --output certs.pem pipeline.yaml

Example pipeline.yaml:
  - set-fetch-options:
      - user-agent:TSL-Tool/1.0
      - timeout:60s
  - load:
      - https://ec.europa.eu/tools/lotl/eu-lotl.xml
  - select:
      - reference-depth:2
  - transform:
      - embedded:tsl-to-html.xslt
      - /var/www/html/tsl
      - html
  - generate_index:
      - /var/www/html/tsl
      - "EU Trust Lists"

See: https://github.com/sirosfoundation/g119612

`, prog, prog, prog)
}

func main() {
	showHelp := flag.Bool("help", false, "Show help message")
	showVersion := flag.Bool("version", false, "Show version information")
	logLevel := flag.String("log-level", "info", "Logging level: debug, info, warn, error")
	logFormat := flag.String("log-format", "text", "Logging format: text or json")
	outputFile := flag.String("output", "", "Write certificate pool PEM to file")

	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("tsl-tool version %s\n", Version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing pipeline YAML file argument")
		usage()
		os.Exit(1)
	}

	pipelineFile := args[0]

	// Configure logging
	level := parseLogLevel(*logLevel)
	var logger logging.Logger
	if *logFormat == "json" {
		logger = logging.JSONLogger(level)
	} else {
		logger = logging.NewLogger(level)
	}

	logger.Info("Starting tsl-tool",
		logging.F("version", Version),
		logging.F("pipeline", pipelineFile))

	// Load the pipeline from YAML file
	pl, err := pipeline.NewPipeline(pipelineFile)
	if err != nil {
		logger.Error("Failed to load pipeline",
			logging.F("file", pipelineFile),
			logging.F("error", err))
		os.Exit(1)
	}

	// Set the logger on the pipeline
	pl = pl.WithLogger(logger)

	logger.Info("Loaded pipeline",
		logging.F("steps", len(pl.Pipes)))

	// Create initial context
	ctx := pipeline.NewContext()

	// Process the pipeline
	resultCtx, err := pl.Process(ctx)
	if err != nil {
		logger.Error("Pipeline processing failed",
			logging.F("error", err))
		os.Exit(1)
	}

	// Log results
	tslCount := 0
	if resultCtx.TSLs != nil {
		tslCount = resultCtx.TSLs.Size()
	}

	logger.Info("Pipeline completed successfully",
		logging.F("tsl_count", tslCount),
		logging.F("cert_pool_exists", resultCtx.CertPool != nil))

	// Write certificate pool to file if requested
	if *outputFile != "" && resultCtx.TSLs != nil {
		// Get all certs from TSLs and write them
		var pemData []byte
		var certCount int
		tsls := resultCtx.TSLs.ToSlice()
		for _, tsl := range tsls {
			if tsl == nil {
				continue
			}
			// Extract certificates from TSL
			tsl.WithTrustServices(func(tsp *etsi119612.TSPType, svc *etsi119612.TSPServiceType) {
				svc.WithCertificates(func(cert *x509.Certificate) {
					block := &pem.Block{
						Type:  "CERTIFICATE",
						Bytes: cert.Raw,
					}
					pemData = append(pemData, pem.EncodeToMemory(block)...)
					certCount++
				})
			})
		}

		if len(pemData) > 0 {
			if err := os.WriteFile(*outputFile, pemData, 0644); err != nil {
				logger.Error("Failed to write certificate pool",
					logging.F("file", *outputFile),
					logging.F("error", err))
				os.Exit(1)
			}
			logger.Info("Wrote certificate pool",
				logging.F("file", *outputFile),
				logging.F("bytes", len(pemData)),
				logging.F("certificates", certCount))
		} else {
			logger.Warn("No certificates to write",
				logging.F("file", *outputFile))
		}
	}

	// Report some stats if we have them
	if resultCtx.TSLTrees != nil && !resultCtx.TSLTrees.IsEmpty() {
		trees := resultCtx.TSLTrees.ToSlice()
		for i, tree := range trees {
			if tree == nil {
				continue
			}
			logger.Debug("TSL tree summary",
				logging.F("index", i),
				logging.F("depth", tree.Depth()),
				logging.F("count", tree.Count()))
		}
	}

	logger.Info("tsl-tool completed",
		logging.F("status", "success"))
}
