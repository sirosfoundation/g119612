package etsi119602

import (
	"fmt"
	"io"
	"net/http"
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
// File paths are detected by the "file://" prefix or absence of "://".
func FetchLoTE(location string, opts *FetchOptions) (*ListOfTrustedEntities, error) {
	if strings.HasPrefix(location, "file://") {
		path := strings.TrimPrefix(location, "file://")
		return ParseLoTEFromFile(path)
	}
	if !strings.Contains(location, "://") {
		return ParseLoTEFromFile(location)
	}
	return fetchLoTEFromURL(location, opts)
}

// FetchAndVerifyLoTE fetches a JWS-signed LoTE, verifies the signature, and returns the payload.
func FetchAndVerifyLoTE(location string, opts *FetchOptions, verifier JWSVerifier) (*ListOfTrustedEntities, error) {
	data, err := fetchRawFromURL(location, opts)
	if err != nil {
		return nil, err
	}
	payload, err := verifier.Verify(string(data))
	if err != nil {
		return nil, fmt.Errorf("JWS verification failed for %s: %w", location, err)
	}
	return ParseLoTE(payload)
}

func fetchLoTEFromURL(url string, opts *FetchOptions) (*ListOfTrustedEntities, error) {
	body, err := fetchRawFromURL(url, opts)
	if err != nil {
		return nil, err
	}
	return ParseLoTE(body)
}

func fetchRawFromURL(url string, opts *FetchOptions) ([]byte, error) {
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
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Set("Accept", "application/json")
	if opts != nil && opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch LoTE from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching %s", resp.StatusCode, url)
	}

	// Warn-level check: reject responses that are clearly not JSON or JWS
	ct := resp.Header.Get("Content-Type")
	if ct != "" {
		ct = strings.ToLower(strings.SplitN(ct, ";", 2)[0])
		ct = strings.TrimSpace(ct)
		if ct != "application/json" && ct != "application/jose" &&
			ct != "text/plain" && ct != "application/octet-stream" && ct != "" {
			return nil, fmt.Errorf("unexpected Content-Type %q fetching %s", ct, url)
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read LoTE response from %s: %w", url, err)
	}

	return body, nil
}
