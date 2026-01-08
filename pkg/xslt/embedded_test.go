package xslt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	files, err := List()
	require.NoError(t, err, "List() should not return an error")
	require.NotEmpty(t, files, "List() should return at least one XSLT file")

	// Check that all returned files have .xslt extension
	for _, file := range files {
		assert.True(t, strings.HasSuffix(file, ".xslt"),
			"File %s should have .xslt extension", file)
	}

	// Verify that tsl-to-html.xslt is in the list (it's the main embedded XSLT)
	found := false
	for _, file := range files {
		if file == "tsl-to-html.xslt" {
			found = true
			break
		}
	}
	assert.True(t, found, "List() should include tsl-to-html.xslt")
}

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		contains string // substring to verify in content
	}{
		{
			name:     "Valid XSLT file",
			filename: "tsl-to-html.xslt",
			wantErr:  false,
			contains: "xsl:stylesheet",
		},
		{
			name:     "Non-existent file",
			filename: "nonexistent.xslt",
			wantErr:  true,
			contains: "",
		},
		{
			name:     "Empty filename",
			filename: "",
			wantErr:  true,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := Get(tt.filename)

			if tt.wantErr {
				assert.Error(t, err, "Get() should return an error for %s", tt.filename)
				assert.Nil(t, content, "Content should be nil on error")
			} else {
				assert.NoError(t, err, "Get() should not return an error for %s", tt.filename)
				assert.NotEmpty(t, content, "Content should not be empty")

				if tt.contains != "" {
					assert.Contains(t, string(content), tt.contains,
						"Content should contain '%s'", tt.contains)
				}

				// Verify it's valid XML by checking for XML declaration or root element
				contentStr := string(content)
				assert.True(t,
					strings.Contains(contentStr, "<?xml") || strings.Contains(contentStr, "<xsl:stylesheet"),
					"Content should be valid XML/XSLT")
			}
		})
	}
}

func TestGetTSLToHTMLXSLT(t *testing.T) {
	// Specific test for the main embedded XSLT
	content, err := Get("tsl-to-html.xslt")
	require.NoError(t, err, "Should be able to get tsl-to-html.xslt")
	require.NotEmpty(t, content, "Content should not be empty")

	contentStr := string(content)

	// Verify key XSLT elements
	assert.Contains(t, contentStr, "xsl:stylesheet", "Should contain xsl:stylesheet")
	assert.Contains(t, contentStr, "xsl:template", "Should contain xsl:template")
	assert.Contains(t, contentStr, "TrustServiceStatusList", "Should reference TSL elements")
	assert.Contains(t, contentStr, "PicoCSS", "Should reference PicoCSS styling")

	// Verify namespace declarations
	assert.Contains(t, contentStr, "xmlns:xsl", "Should declare xsl namespace")
	assert.Contains(t, contentStr, "xmlns:tsl", "Should declare tsl namespace")
}

func TestPath(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "Simple filename",
			filename: "tsl-to-html.xslt",
			want:     "embedded:tsl-to-html.xslt",
		},
		{
			name:     "Empty string",
			filename: "",
			want:     "embedded:",
		},
		{
			name:     "Filename with path",
			filename: "path/to/file.xslt",
			want:     "embedded:path/to/file.xslt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Path(tt.filename)
			assert.Equal(t, tt.want, got, "Path() should return correct embedded path")
		})
	}
}

func TestIsEmbeddedPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Valid embedded path",
			path: "embedded:tsl-to-html.xslt",
			want: true,
		},
		{
			name: "Embedded path with directory",
			path: "embedded:subdir/file.xslt",
			want: true,
		},
		{
			name: "Regular file path",
			path: "/path/to/file.xslt",
			want: false,
		},
		{
			name: "Relative file path",
			path: "./file.xslt",
			want: false,
		},
		{
			name: "Empty string",
			path: "",
			want: false,
		},
		{
			name: "Short string",
			path: "embedded",
			want: false,
		},
		{
			name: "Embedded without colon",
			path: "embedded",
			want: false,
		},
		{
			name: "URL",
			path: "http://example.com/file.xslt",
			want: false,
		},
		{
			name: "Embedded prefix in middle",
			path: "/path/embedded:file.xslt",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEmbeddedPath(tt.path)
			assert.Equal(t, tt.want, got,
				"IsEmbeddedPath(%q) should return %v", tt.path, tt.want)
		})
	}
}

func TestExtractNameFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "Valid embedded path",
			path: "embedded:tsl-to-html.xslt",
			want: "tsl-to-html.xslt",
		},
		{
			name: "Embedded path with directory",
			path: "embedded:subdir/file.xslt",
			want: "subdir/file.xslt",
		},
		{
			name: "Regular file path",
			path: "/path/to/file.xslt",
			want: "/path/to/file.xslt",
		},
		{
			name: "Relative file path",
			path: "./file.xslt",
			want: "./file.xslt",
		},
		{
			name: "Empty string",
			path: "",
			want: "",
		},
		{
			name: "Just embedded prefix without filename",
			path: "embedded:",
			want: "embedded:", // Not recognized as embedded path (needs filename after colon)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractNameFromPath(tt.path)
			assert.Equal(t, tt.want, got,
				"ExtractNameFromPath(%q) should return %q", tt.path, tt.want)
		})
	}
}

// TestListAndGetConsistency verifies that all files returned by List()
// can be successfully retrieved with Get()
func TestListAndGetConsistency(t *testing.T) {
	files, err := List()
	require.NoError(t, err, "List() should not return an error")
	require.NotEmpty(t, files, "List() should return at least one file")

	for _, filename := range files {
		t.Run("Get_"+filename, func(t *testing.T) {
			content, err := Get(filename)
			assert.NoError(t, err, "Should be able to get file %s", filename)
			assert.NotEmpty(t, content, "Content for %s should not be empty", filename)
		})
	}
}

// TestPathAndExtractRoundTrip verifies that Path() and ExtractNameFromPath()
// are inverse operations
func TestPathAndExtractRoundTrip(t *testing.T) {
	testFilenames := []string{
		"tsl-to-html.xslt",
		"file.xslt",
		"subdir/file.xslt",
	}

	for _, filename := range testFilenames {
		t.Run(filename, func(t *testing.T) {
			embedded := Path(filename)
			extracted := ExtractNameFromPath(embedded)
			assert.Equal(t, filename, extracted,
				"Path() and ExtractNameFromPath() should be inverse operations")
			assert.True(t, IsEmbeddedPath(embedded),
				"Path() result should be recognized as embedded path")
		})
	}
}

// BenchmarkGet benchmarks the Get function
func BenchmarkGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Get("tsl-to-html.xslt")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIsEmbeddedPath benchmarks the IsEmbeddedPath function
func BenchmarkIsEmbeddedPath(b *testing.B) {
	path := "embedded:tsl-to-html.xslt"
	for i := 0; i < b.N; i++ {
		_ = IsEmbeddedPath(path)
	}
}

// BenchmarkExtractNameFromPath benchmarks the ExtractNameFromPath function
func BenchmarkExtractNameFromPath(b *testing.B) {
	path := "embedded:tsl-to-html.xslt"
	for i := 0; i < b.N; i++ {
		_ = ExtractNameFromPath(path)
	}
}
