// Package pipeline provides format detection utilities for trust lists.
package pipeline

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirosfoundation/g119612/pkg/logging"
)

// Format represents the trust list format.
type Format string

const (
	// FormatTSL represents ETSI TS 119 612 Trust Service List (XML).
	FormatTSL Format = "tsl"
	// FormatLoTE represents ETSI TS 119 602 List of Trusted Entities (JSON).
	FormatLoTE Format = "lote"
	// FormatUnknown represents an unrecognized format.
	FormatUnknown Format = ""
)

// DetectFormat determines the format of trust list data based on multiple signals:
// 1. File extension from the location
// 2. Content-Type header (for HTTP responses)
// 3. Content probing (first non-whitespace byte)
//
// Parameters:
//   - location: URL or file path
//   - contentType: HTTP Content-Type header value (can be empty)
//   - data: Raw content bytes (can be nil for extension-only detection)
//
// Returns the detected Format.
func DetectFormat(location, contentType string, data []byte) Format {
	// 1. Try extension-based detection
	if format := detectFormatByExtension(location); format != FormatUnknown {
		return format
	}

	// 2. Try Content-Type header
	if format := detectFormatByContentType(contentType); format != FormatUnknown {
		return format
	}

	// 3. Try content probing
	if format := detectFormatByContent(data); format != FormatUnknown {
		return format
	}

	return FormatUnknown
}

// detectFormatByExtension detects format from file extension.
func detectFormatByExtension(location string) Format {
	lower := strings.ToLower(location)

	// Remove query string if present
	if idx := strings.Index(lower, "?"); idx != -1 {
		lower = lower[:idx]
	}

	switch {
	case strings.HasSuffix(lower, ".xml"):
		return FormatTSL
	case strings.HasSuffix(lower, ".json"):
		return FormatLoTE
	case strings.HasSuffix(lower, ".jws"):
		return FormatLoTE // JWS-signed LoTE
	default:
		return FormatUnknown
	}
}

// detectFormatByContentType detects format from HTTP Content-Type header.
func detectFormatByContentType(contentType string) Format {
	if contentType == "" {
		return FormatUnknown
	}

	lower := strings.ToLower(contentType)

	switch {
	case strings.Contains(lower, "application/xml"),
		strings.Contains(lower, "text/xml"):
		return FormatTSL
	case strings.Contains(lower, "application/json"),
		strings.Contains(lower, "application/jose"): // JWS compact serialization
		return FormatLoTE
	default:
		return FormatUnknown
	}
}

// detectFormatByContent probes the content to detect format.
func detectFormatByContent(data []byte) Format {
	if len(data) == 0 {
		return FormatUnknown
	}

	// Skip leading whitespace and BOM
	trimmed := bytes.TrimLeft(data, " \t\n\r\xef\xbb\xbf")
	if len(trimmed) == 0 {
		return FormatUnknown
	}

	firstByte := trimmed[0]
	switch firstByte {
	case '<':
		return FormatTSL // XML
	case '{', '[':
		return FormatLoTE // JSON
	case 'e':
		// Could be JWS compact serialization starting with "eyJ" (base64url of "{")
		if len(trimmed) >= 3 && string(trimmed[:3]) == "eyJ" {
			return FormatLoTE
		}
		return FormatUnknown
	default:
		return FormatUnknown
	}
}

// FetchRaw fetches raw content from a URL or file path, returning the data,
// Content-Type header (for HTTP), and any error.
func FetchRaw(location string, opts *FetchRawOptions) ([]byte, string, error) {
	// Handle file:// scheme or local paths
	if strings.HasPrefix(location, "file://") {
		path := strings.TrimPrefix(location, "file://")
		data, err := os.ReadFile(path)
		return data, "", err
	}

	// No scheme = local file
	if !strings.Contains(location, "://") {
		data, err := os.ReadFile(location)
		return data, "", err
	}

	// HTTP/HTTPS fetch
	return fetchRawFromURL(location, opts)
}

// FetchRawOptions configures raw content fetching.
type FetchRawOptions struct {
	UserAgent string
	Timeout   time.Duration
	Accept    string
}

func fetchRawFromURL(url string, opts *FetchRawOptions) ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	if opts != nil && opts.Timeout > 0 {
		client.Timeout = opts.Timeout
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	// Accept both XML and JSON
	accept := "*/*"
	if opts != nil && opts.Accept != "" {
		accept = opts.Accept
	}
	req.Header.Set("Accept", accept)

	if opts != nil && opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", &HTTPError{StatusCode: resp.StatusCode, URL: url}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return data, resp.Header.Get("Content-Type"), nil
}

// HTTPError represents an HTTP error from fetching content.
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	return "HTTP " + http.StatusText(e.StatusCode) + " fetching " + e.URL
}

// LogFormat logs the detected format for debugging.
func LogFormat(logger logging.Logger, location string, format Format) {
	if logger == nil {
		return
	}
	formatStr := "unknown"
	switch format {
	case FormatTSL:
		formatStr = "TSL (XML)"
	case FormatLoTE:
		formatStr = "LoTE (JSON)"
	}
	logger.Debug("Detected format", logging.F("location", location), logging.F("format", formatStr))
}
