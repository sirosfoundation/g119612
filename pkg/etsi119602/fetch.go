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

func fetchLoTEFromURL(url string, opts *FetchOptions) (*ListOfTrustedEntities, error) {
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read LoTE response from %s: %w", url, err)
	}

	return ParseLoTE(body)
}
