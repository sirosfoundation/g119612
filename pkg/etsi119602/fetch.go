package etsi119602

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FetchOptions configures how LoTEs are fetched.
type FetchOptions struct {
	// UserAgent is sent in the User-Agent header.
	UserAgent string

	// Timeout is the HTTP request timeout.
	Timeout time.Duration

	// HTTPClient allows injecting a custom client (e.g., for testing).
	HTTPClient *http.Client
}

// JWSVerifier verifies JWS compact serializations and returns the payload.
type JWSVerifier interface {
	Verify(compact string) ([]byte, error)
}

// FetchLoTE fetches and parses a LoTE from a URL or file path.
// Automatically detects JSON vs XML format based on file extension
// and content. File paths are detected by the "file://" prefix or
// absence of "://".
func FetchLoTE(location string, opts *FetchOptions) (*ListOfTrustedEntities, error) {
	if strings.HasPrefix(location, "file://") {
		path := strings.TrimPrefix(location, "file://")
		return parseLoTEFileAutoDetect(path)
	}
	if !strings.Contains(location, "://") {
		return parseLoTEFileAutoDetect(location)
	}
	return fetchLoTEFromURL(location, opts)
}

// FetchLoTEXML fetches and parses a LoTE from an XML URL or file path.
func FetchLoTEXML(location string, opts *FetchOptions) (*ListOfTrustedEntities, error) {
	if strings.HasPrefix(location, "file://") {
		path := strings.TrimPrefix(location, "file://")
		return ParseLoTEXMLFromFile(path)
	}
	if !strings.Contains(location, "://") {
		return ParseLoTEXMLFromFile(location)
	}
	data, err := fetchRawFromURL(location, opts, "application/xml")
	if err != nil {
		return nil, err
	}
	return ParseLoTEXML(data)
}

// FetchLoTL fetches and parses a LoTL from a URL or file path.
// Automatically detects JSON vs XML format.
func FetchLoTL(location string, opts *FetchOptions) (*ListOfTrustedLists, error) {
	if strings.HasPrefix(location, "file://") {
		path := strings.TrimPrefix(location, "file://")
		return parseLoTLFileAutoDetect(path)
	}
	if !strings.Contains(location, "://") {
		return parseLoTLFileAutoDetect(location)
	}
	return fetchLoTLFromURL(location, opts)
}

// FetchLoTLXML fetches and parses a LoTL from an XML URL or file path.
func FetchLoTLXML(location string, opts *FetchOptions) (*ListOfTrustedLists, error) {
	if strings.HasPrefix(location, "file://") {
		path := strings.TrimPrefix(location, "file://")
		return ParseLoTLXMLFromFile(path)
	}
	if !strings.Contains(location, "://") {
		return ParseLoTLXMLFromFile(location)
	}
	data, err := fetchRawFromURL(location, opts, "application/xml")
	if err != nil {
		return nil, err
	}
	return ParseLoTLXML(data)
}

// FetchAndVerifyLoTE fetches a JWS-signed LoTE, verifies the signature, and returns the payload.
func FetchAndVerifyLoTE(location string, opts *FetchOptions, verifier JWSVerifier) (*ListOfTrustedEntities, error) {
	data, err := fetchRawFromURL(location, opts, "application/jose, application/json")
	if err != nil {
		return nil, err
	}
	payload, err := verifier.Verify(string(data))
	if err != nil {
		return nil, fmt.Errorf("JWS verification failed for %s: %w", location, err)
	}
	return ParseLoTE(payload)
}

// FetchRaw fetches raw bytes from a URL or file path without parsing.
// For file paths, reads the file directly. For URLs, fetches with Accept: */*.
func FetchRaw(location string, opts *FetchOptions) ([]byte, error) {
	if strings.HasPrefix(location, "file://") {
		return os.ReadFile(strings.TrimPrefix(location, "file://"))
	}
	if !strings.Contains(location, "://") {
		return os.ReadFile(location)
	}
	return fetchRawFromURL(location, opts, "*/*")
}

// isXMLLocation returns true if the location (URL or file path) looks like XML.
func isXMLLocation(location string) bool {
	// Strip query/fragment for URL analysis
	loc := location
	if idx := strings.IndexAny(loc, "?#"); idx >= 0 {
		loc = loc[:idx]
	}
	ext := strings.ToLower(filepath.Ext(loc))
	return ext == ".xml"
}

// isXMLContent detects XML content by checking for the XML declaration or
// a root element starting with '<'.
func isXMLContent(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return bytes.HasPrefix(trimmed, []byte("<?xml")) || bytes.HasPrefix(trimmed, []byte("<"))
}

// parseLoTEFileAutoDetect loads a LoTE from a file, auto-detecting JSON vs XML.
func parseLoTEFileAutoDetect(path string) (*ListOfTrustedEntities, error) {
	if isXMLLocation(path) {
		return ParseLoTEXMLFromFile(path)
	}
	return ParseLoTEFromFile(path)
}

// parseLoTLFileAutoDetect loads a LoTL from a file, auto-detecting JSON vs XML.
func parseLoTLFileAutoDetect(path string) (*ListOfTrustedLists, error) {
	if isXMLLocation(path) {
		return ParseLoTLXMLFromFile(path)
	}
	return ParseLoTLFromFile(path)
}

func fetchLoTEFromURL(url string, opts *FetchOptions) (*ListOfTrustedEntities, error) {
	accept := "application/json"
	if isXMLLocation(url) {
		accept = "application/xml"
	}
	body, contentType, err := fetchRawWithContentType(url, opts, accept)
	if err != nil {
		return nil, err
	}
	if isXMLContentType(contentType) || (contentType == "" && isXMLContent(body)) {
		return ParseLoTEXML(body)
	}
	return ParseLoTE(body)
}

func fetchLoTLFromURL(url string, opts *FetchOptions) (*ListOfTrustedLists, error) {
	accept := "application/json"
	if isXMLLocation(url) {
		accept = "application/xml"
	}
	body, contentType, err := fetchRawWithContentType(url, opts, accept)
	if err != nil {
		return nil, err
	}
	if isXMLContentType(contentType) || (contentType == "" && isXMLContent(body)) {
		return ParseLoTLXML(body)
	}
	return ParseLoTL(body)
}

func isXMLContentType(ct string) bool {
	return ct == "application/xml" || ct == "text/xml"
}

func fetchRawFromURL(url string, opts *FetchOptions, accept string) ([]byte, error) {
	body, _, err := fetchRawWithContentType(url, opts, accept)
	return body, err
}

func fetchRawWithContentType(url string, opts *FetchOptions, accept string) ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	if opts != nil {
		if opts.HTTPClient != nil {
			client = opts.HTTPClient
		} else if opts.Timeout > 0 {
			client.Timeout = opts.Timeout
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Set("Accept", accept)
	if opts != nil && opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d fetching %s", resp.StatusCode, url)
	}

	ct := ""
	if raw := resp.Header.Get("Content-Type"); raw != "" {
		ct = strings.ToLower(strings.TrimSpace(strings.SplitN(raw, ";", 2)[0]))
	}

	// Reject clearly wrong content types
	allowedTypes := map[string]bool{
		"application/json":         true,
		"application/jose":         true,
		"application/xml":          true,
		"text/xml":                 true,
		"text/plain":               true,
		"application/octet-stream": true,
		"":                         true,
	}
	if !allowedTypes[ct] {
		return nil, "", fmt.Errorf("unexpected Content-Type %q fetching %s", ct, url)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response from %s: %w", url, err)
	}

	return body, ct, nil
}
