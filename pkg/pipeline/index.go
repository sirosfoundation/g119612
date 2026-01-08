// Package pipeline provides a pipeline framework for processing Trust Status Lists (TSLs).
package pipeline

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

//go:embed templates/index.html
var indexHTMLTemplate string

//go:embed templates/index.css
var indexCSS string

//go:embed templates/index.js
var indexJavaScript string

// TSLIndexEntry represents a single Trust Service List entry in the index
type TSLIndexEntry struct {
	Filename     string // Name of the HTML file
	Title        string // Title of the TSL (usually country name)
	SchemeType   string // Type of the TSL scheme
	Territory    string // Territory code
	Sequence     string // Sequence number
	IssueDate    string // Issue date of the TSL
	NextUpdate   string // Next update date
	URL          string // Link to the HTML file
	TrustService int    // Number of trust services in the TSL
}

// GenerateIndex creates an index.html file in the specified directory.
// The index page lists all TSL HTML files in the directory with metadata and links.
// The index uses PicoCSS for styling to match the TSL HTML files.
//
// Arguments:
//   - arg[0]: Directory path containing TSL HTML files
//   - arg[1]: (Optional) Title for the index page (default: "Trust Service Lists Index")
//
// Example usage in pipeline YAML:
//
//   - generate_index:
//   - /path/to/output/directory
//   - "EU Trust Lists - Index"
func GenerateIndex(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return ctx, fmt.Errorf("missing required directory path argument")
	}

	// Parse arguments
	dirPath := args[0]
	title := "Trust Service Lists Index"
	if len(args) >= 2 {
		title = args[1]
	}

	// Check if the directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ctx, fmt.Errorf("directory %s does not exist", dirPath)
		}
		return ctx, fmt.Errorf("error accessing directory %s: %w", dirPath, err)
	}
	if !info.IsDir() {
		return ctx, fmt.Errorf("%s is not a directory", dirPath)
	}

	// Find all HTML files in the directory
	entries, err := findTSLHtmlFiles(dirPath)
	if err != nil {
		return ctx, fmt.Errorf("failed to read directory: %w", err)
	}

	if len(entries) == 0 {
		return ctx, fmt.Errorf("no TSL HTML files found in %s", dirPath)
	}

	// Generate the index.html file
	err = generateIndexHTML(dirPath, entries, title)
	if err != nil {
		return ctx, fmt.Errorf("failed to generate index.html: %w", err)
	}

	return ctx, nil
}

// findTSLHtmlFiles scans a directory for TSL HTML files and extracts metadata from them
func findTSLHtmlFiles(dirPath string) ([]TSLIndexEntry, error) {
	var entries []TSLIndexEntry

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-HTML files
		if d.IsDir() || filepath.Ext(path) != ".html" || filepath.Base(path) == "index.html" {
			return nil
		}

		// Get the relative path for the URL
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// Extract metadata from the HTML file
		entry, err := extractMetadataFromHTML(path, relPath)
		if err != nil {
			// Skip files that don't appear to be TSL HTML files
			return nil
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort entries by territory code
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Territory < entries[j].Territory
	})

	return entries, nil
}

// extractMetadataFromHTML reads a TSL HTML file and extracts metadata for the index
func extractMetadataFromHTML(filePath, relPath string) (TSLIndexEntry, error) {
	entry := TSLIndexEntry{
		Filename: filepath.Base(filePath),
		URL:      relPath,
	}

	// Read the HTML file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return entry, err
	}

	// Parse the HTML document
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return entry, err
	}

	// Extract title
	entry.Title = doc.Find("title").Text()

	// Extract territory from the specific element if available
	if territoryText := doc.Find(".tsl-meta:contains('Territory')").Text(); territoryText != "" {
		// Try to extract the territory code (usually formatted as "Territory: XX")
		if idx := strings.Index(territoryText, "Territory:"); idx != -1 {
			territory := strings.TrimSpace(territoryText[idx+len("Territory:"):])
			// If we got a territory code (usually 2 characters), use it
			if len(territory) == 2 {
				entry.Territory = territory
			} else {
				// Try to get territory from the title
				parts := strings.Split(entry.Title, " - ")
				if len(parts) > 0 {
					entry.Territory = strings.TrimSpace(parts[0])
				}
			}
		}
	} else {
		// Try to extract from title (common format: "[TERRITORY] - Trust Service Status List")
		parts := strings.Split(entry.Title, " - ")
		if len(parts) > 0 {
			entry.Territory = strings.TrimSpace(parts[0])
		}
	}

	// Extract TSL type
	entry.SchemeType = doc.Find(".tsl-meta code").First().Text()

	// Extract sequence number
	seq := doc.Find(".tsl-meta:contains('TSL Sequence')").Text()
	if idx := strings.Index(seq, "TSL Sequence #:"); idx != -1 {
		parts := strings.Split(seq[idx:], "|")
		if len(parts) > 0 {
			entry.Sequence = strings.TrimSpace(strings.TrimPrefix(parts[0], "TSL Sequence #:"))
		}
	}

	// Extract issue date
	issue := doc.Find(".tsl-meta:contains('Issue Date')").Text()
	if idx := strings.Index(issue, "Issue Date:"); idx != -1 {
		parts := strings.Split(issue[idx:], "|")
		if len(parts) > 0 {
			entry.IssueDate = strings.TrimSpace(strings.TrimPrefix(parts[0], "Issue Date:"))
		}
	}

	// Extract next update date
	next := doc.Find(".tsl-meta:contains('Next Update')").Text()
	if idx := strings.Index(next, "Next Update:"); idx != -1 {
		parts := strings.Split(next[idx:], "|")
		if len(parts) > 0 {
			entry.NextUpdate = strings.TrimSpace(strings.TrimPrefix(parts[0], "Next Update:"))
		}
	}

	// Count trust services
	entry.TrustService = doc.Find(".service-card").Length()

	return entry, nil
}

// generateIndexHTML creates an index.html file with links to all TSL HTML files using embedded templates
func generateIndexHTML(dirPath string, entries []TSLIndexEntry, title string) error {
	// Prepare template data
	data := struct {
		Title         string
		Entries       []TSLIndexEntry
		GeneratedDate string
		CSS           template.CSS
		JavaScript    template.JS
	}{
		Title:         title,
		Entries:       entries,
		GeneratedDate: time.Now().Format("2006-01-02"),
		CSS:           template.CSS(indexCSS),
		JavaScript:    template.JS(indexJavaScript),
	}

	// Parse and execute the template
	tmpl, err := template.New("index").Parse(indexHTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create the index.html file
	file, err := os.Create(filepath.Join(dirPath, "index.html"))
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer file.Close()

	// Execute the template and write to the file
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func init() {
	// Register the GenerateIndex function
	RegisterFunction("generate_index", GenerateIndex)
}
