// Package validation provides input validation and sanitization functions for Go-Trust.package validation

package validation

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// URLValidationOptions defines options for URL validation
type URLValidationOptions struct {
	// AllowedSchemes restricts URLs to specific schemes (e.g., http, https, file)
	AllowedSchemes []string
	// RequireAbsoluteURL requires the URL to be absolute (have a scheme)
	RequireAbsoluteURL bool
	// AllowFileURLs allows file:// URLs
	AllowFileURLs bool
}

// DefaultURLOptions returns default URL validation options
func DefaultURLOptions() URLValidationOptions {
	return URLValidationOptions{
		AllowedSchemes:     []string{"http", "https"},
		RequireAbsoluteURL: true,
		AllowFileURLs:      false,
	}
}

// TSLURLOptions returns URL validation options suitable for TSL loading
func TSLURLOptions() URLValidationOptions {
	return URLValidationOptions{
		AllowedSchemes:     []string{"http", "https", "file"},
		RequireAbsoluteURL: true,
		AllowFileURLs:      true,
	}
}

// ValidateURL validates a URL string according to the provided options
func ValidateURL(rawURL string, opts URLValidationOptions) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check if absolute URL is required
	if opts.RequireAbsoluteURL && parsedURL.Scheme == "" {
		return fmt.Errorf("URL must be absolute (include scheme)")
	}

	// Check allowed schemes
	if len(opts.AllowedSchemes) > 0 && parsedURL.Scheme != "" {
		allowed := false
		for _, scheme := range opts.AllowedSchemes {
			if strings.EqualFold(parsedURL.Scheme, scheme) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("URL scheme '%s' not allowed (allowed: %v)", parsedURL.Scheme, opts.AllowedSchemes)
		}
	}

	// Special handling for file:// URLs
	if strings.EqualFold(parsedURL.Scheme, "file") {
		if !opts.AllowFileURLs {
			return fmt.Errorf("file:// URLs are not allowed")
		}
		// Validate the file path component
		path := strings.TrimPrefix(rawURL, "file://")
		if err := ValidateFilePath(path); err != nil {
			return fmt.Errorf("invalid file path in URL: %w", err)
		}
	}

	// Check for suspicious patterns that might indicate path traversal attempts
	if strings.Contains(parsedURL.Path, "..") {
		return fmt.Errorf("URL path contains '..' which could indicate path traversal attempt")
	}

	return nil
}

// ValidateFilePath validates a file path for security issues
func ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Clean the path to resolve any .., ., or // sequences
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		// Verify that after cleaning, we haven't escaped expected directories
		// This is a basic check - for production, consider more sophisticated validation
		if strings.HasPrefix(cleanPath, "..") {
			return fmt.Errorf("path traversal detected: path attempts to escape allowed directories")
		}
	}

	// Check for null bytes (security issue in some contexts)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null bytes")
	}

	// Check for potentially dangerous patterns
	dangerousPatterns := []string{
		"/etc/passwd",
		"/etc/shadow",
		"c:\\windows\\system32",
		"c:\\windows\\system",
	}

	lowerPath := strings.ToLower(cleanPath)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerPath, pattern) {
			return fmt.Errorf("path contains suspicious pattern: %s", pattern)
		}
	}

	return nil
}

// SanitizeFilePath sanitizes a file path by cleaning it and converting to absolute path
func SanitizeFilePath(path string) string {
	// Clean the path to resolve ., .., and // sequences
	cleanPath := filepath.Clean(path)

	// Convert to absolute path if possible
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		// If we can't get absolute path, return cleaned path
		return cleanPath
	}

	return absPath
}

// ValidateConfigPath validates a configuration file path
func ValidateConfigPath(path string) error {
	if err := ValidateFilePath(path); err != nil {
		return fmt.Errorf("invalid config file path: %w", err)
	}

	// Additional checks for config files
	if !strings.HasSuffix(strings.ToLower(path), ".yaml") &&
		!strings.HasSuffix(strings.ToLower(path), ".yml") {
		return fmt.Errorf("config file must have .yaml or .yml extension")
	}

	return nil
}

// ValidateXSLTPath validates an XSLT stylesheet path
func ValidateXSLTPath(path string) error {
	// Check if it's an embedded XSLT (special case)
	if strings.HasPrefix(path, "embedded:") {
		embeddedName := strings.TrimPrefix(path, "embedded:")
		if embeddedName == "" {
			return fmt.Errorf("embedded XSLT name cannot be empty")
		}
		if strings.Contains(embeddedName, "..") || strings.Contains(embeddedName, "/") {
			return fmt.Errorf("embedded XSLT name cannot contain path separators")
		}
		return nil
	}

	// Regular file path validation
	if err := ValidateFilePath(path); err != nil {
		return fmt.Errorf("invalid XSLT file path: %w", err)
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".xslt" && ext != ".xsl" {
		return fmt.Errorf("XSLT file must have .xslt or .xsl extension")
	}

	return nil
}

// ValidateOutputDirectory validates an output directory path
func ValidateOutputDirectory(path string) error {
	if err := ValidateFilePath(path); err != nil {
		return fmt.Errorf("invalid output directory: %w", err)
	}

	// Ensure it's not a root directory or system directory
	cleanPath := filepath.Clean(path)
	if cleanPath == "/" || cleanPath == "C:\\" || cleanPath == "c:\\" {
		return fmt.Errorf("cannot use root directory as output directory")
	}

	systemDirs := []string{"/etc", "/sys", "/proc", "/dev", "c:\\windows", "c:\\program files"}
	lowerPath := strings.ToLower(cleanPath)
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(lowerPath, sysDir) {
			return fmt.Errorf("cannot use system directory as output directory")
		}
	}

	return nil
}
