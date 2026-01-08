package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateIndex(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tsl-index-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a subdirectory for the HTML files
	htmlDir := filepath.Join(tempDir, "html")
	require.NoError(t, os.MkdirAll(htmlDir, 0755))

	// Create some sample HTML files that resemble TSL files
	createSampleTSLHTML(t, htmlDir, "SE-TL.html", "Sweden", "SE", "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUlistofthelists", "42", "2025-09-15", "2025-12-15", 5)
	createSampleTSLHTML(t, htmlDir, "DE-TL.html", "Germany", "DE", "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric", "73", "2025-09-10", "2025-12-10", 12)
	createSampleTSLHTML(t, htmlDir, "FI-TL.html", "Finland", "FI", "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric", "31", "2025-09-20", "2025-12-20", 3)

	// Create an empty file that should be ignored
	emptyFile := filepath.Join(htmlDir, "empty.html")
	require.NoError(t, os.WriteFile(emptyFile, []byte("<html></html>"), 0644))

	t.Run("Basic Index Generation", func(t *testing.T) {
		ctx := NewContext()

		// Call the GenerateIndex function
		resultCtx, err := GenerateIndex(nil, ctx, htmlDir, "Test TSL Index")
		assert.NoError(t, err)
		assert.NotNil(t, resultCtx)

		// Check if the index.html file was created
		indexPath := filepath.Join(htmlDir, "index.html")
		_, err = os.Stat(indexPath)
		assert.NoError(t, err, "index.html should exist")

		// Read the file content
		content, err := os.ReadFile(indexPath)
		assert.NoError(t, err)

		// Check if the content contains expected elements
		indexHTML := string(content)
		assert.Contains(t, indexHTML, "Test TSL Index")
		// Check for territory codes in the badges
		assert.Contains(t, indexHTML, `<span class="badge badge-country">SE</span>`)
		assert.Contains(t, indexHTML, `<span class="badge badge-country">DE</span>`)
		assert.Contains(t, indexHTML, `<span class="badge badge-country">FI</span>`)
	})

	t.Run("Directory Not Found", func(t *testing.T) {
		ctx := NewContext()

		// Call with non-existent directory
		_, err := GenerateIndex(nil, ctx, filepath.Join(tempDir, "nonexistent"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("Empty Directory", func(t *testing.T) {
		ctx := NewContext()

		// Create an empty directory
		emptyDir := filepath.Join(tempDir, "empty")
		require.NoError(t, os.MkdirAll(emptyDir, 0755))

		// Call with empty directory
		_, err := GenerateIndex(nil, ctx, emptyDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no TSL HTML files found")
	})

	t.Run("Custom Title", func(t *testing.T) {
		ctx := NewContext()
		customTitle := "Custom Index Title"

		// Call with custom title
		_, err := GenerateIndex(nil, ctx, htmlDir, customTitle)
		assert.NoError(t, err)

		// Read the file content
		content, err := os.ReadFile(filepath.Join(htmlDir, "index.html"))
		assert.NoError(t, err)

		// Check if the title is correct
		assert.Contains(t, string(content), customTitle)
	})
}

// Helper function to create sample TSL HTML files for testing
func createSampleTSLHTML(t *testing.T, dirPath, filename, title, territory, schemeType, sequence, issueDate, nextUpdate string, services int) {
	// Create a minimal HTML structure that mimics a TSL HTML file
	htmlContent := `<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + ` - Trust Service Status List</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@1/css/pico.min.css"/>
</head>
<body>
    <main class="container">
        <div class="tsl-meta">
            <p>
                <strong>TSL Sequence #:</strong> ` + sequence + ` | 
                <strong>Issue Date:</strong> ` + issueDate + ` | 
                <strong>Next Update:</strong> ` + nextUpdate + `
            </p>
            <p>
                <strong>TSL Type:</strong> <code>` + schemeType + `</code>
            </p>
            <p>
                <strong>Territory:</strong> ` + territory + `
            </p>
        </div>`

	// Add service cards based on the specified count
	for i := 0; i < services; i++ {
		htmlContent += `
        <article class="service-card">
            <h4>Test Service ` + territory + ` ` + string(rune('A'+i)) + `</h4>
        </article>`
	}

	htmlContent += `
    </main>
</body>
</html>`

	// Write the HTML content to a file
	err := os.WriteFile(filepath.Join(dirPath, filename), []byte(htmlContent), 0644)
	require.NoError(t, err)
}
