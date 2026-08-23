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
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirosfoundation/g119612/pkg/dsig"
	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/pipeline"
	"github.com/sirosfoundation/go-cryptoutil"
	"github.com/sirosfoundation/go-cryptoutil/brainpool"
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
  generate_index   Generate HTML index of TSL files (includes report by default)
  report           Generate standalone pipeline report (deprecated, use generate_index)
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
      # add 'no-report' to skip report.html generation

See: https://github.com/sirosfoundation/g119612

`, prog, prog, prog)
}

// runSelfSignCert issues a self-signed certificate for a key pair that already
// exists in a PKCS#11 token, so the certificate and the signing key cannot
// disagree. See dsig.SelfSignCertificate for why this is not done with openssl.
func runSelfSignCert(pkcs11URI, keyLabel, keyID, certLabel, subject string, days int, outputFile string) error {
	if pkcs11URI == "" {
		return fmt.Errorf("-pkcs11-uri is required with -selfsign-cert")
	}
	if subject == "" {
		return fmt.Errorf("-subject is required with -selfsign-cert")
	}
	if days <= 0 {
		return fmt.Errorf("-days must be positive, got %d", days)
	}

	config := dsig.ExtractPKCS11Config(pkcs11URI)
	if config == nil {
		return fmt.Errorf("invalid PKCS#11 URI")
	}

	cert, pemBytes, err := dsig.SelfSignCertificate(config, dsig.SelfSignedCertOptions{
		KeyLabel:  keyLabel,
		KeyID:     keyID,
		CertLabel: certLabel,
		Subject:   pkix.Name{CommonName: subject},
		Validity:  time.Duration(days) * 24 * time.Hour,
	})
	if err != nil {
		return err
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, pemBytes, 0644); err != nil {
			return fmt.Errorf("failed to write certificate to %s: %w", outputFile, err)
		}
		fmt.Fprintf(os.Stderr, "Wrote certificate to %s\n", outputFile)
	} else {
		if _, err := os.Stdout.Write(pemBytes); err != nil {
			return fmt.Errorf("failed to write certificate: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "Subject:   %s\n", cert.Subject)
	fmt.Fprintf(os.Stderr, "Serial:    %s\n", cert.SerialNumber)
	fmt.Fprintf(os.Stderr, "Not after: %s\n", cert.NotAfter.Format(time.RFC3339))
	if certLabel != "" {
		fmt.Fprintf(os.Stderr, "Stored in token under label %q\n", certLabel)
	}
	return nil
}

func main() {
	showHelp := flag.Bool("help", false, "Show help message")
	showVersion := flag.Bool("version", false, "Show version information")
	logLevel := flag.String("log-level", "info", "Logging level: debug, info, warn, error")
	logFormat := flag.String("log-format", "text", "Logging format: text or json")
	outputFile := flag.String("output", "", "Write certificate pool PEM to file")

	selfSignCert := flag.Bool("selfsign-cert", false, "Issue a self-signed certificate for an existing PKCS#11 key instead of running a pipeline")
	pkcs11URI := flag.String("pkcs11-uri", "", "PKCS#11 URI (with -selfsign-cert)")
	keyLabel := flag.String("key-label", "signing-key", "PKCS#11 key label (with -selfsign-cert)")
	keyID := flag.String("key-id", "01", "PKCS#11 key ID in hex (with -selfsign-cert)")
	certLabel := flag.String("cert-label", "", "Store the certificate in the token under this label (with -selfsign-cert)")
	subject := flag.String("subject", "", "Certificate common name (with -selfsign-cert)")
	days := flag.Int("days", 1095, "Certificate validity in days (with -selfsign-cert)")

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

	if *selfSignCert {
		if err := runSelfSignCert(*pkcs11URI, *keyLabel, *keyID, *certLabel, *subject, *days, *outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
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

	// Create initial context with brainpool crypto extensions
	ext := cryptoutil.New()
	brainpool.Register(ext)
	ctx := pipeline.NewContext()
	ctx.CryptoExt = ext

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
				}, resultCtx.CryptoExt)
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
