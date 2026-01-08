// Package xslt provides embedded XSLT stylesheets for TSL transformations.
//
// This package uses Go's embed directive to include XSLT stylesheets directly in the
// binary, allowing for transformations without external file dependencies. It provides
// convenient access to standard transformation templates that can be used with the
// pipeline package's transform functionality.
package xslt

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed *.xslt
var embeddedXSLT embed.FS

// List returns a list of available embedded XSLT stylesheets.
func List() ([]string, error) {
	var files []string

	entries, err := fs.ReadDir(embeddedXSLT, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded XSLT directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// Get returns the content of a specific embedded XSLT stylesheet.
func Get(name string) ([]byte, error) {
	content, err := embeddedXSLT.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded XSLT file '%s': %w", name, err)
	}
	return content, nil
}

// Path returns a special path identifier for embedded XSLTs that can be used
// with the transform.go pipeline step. The path follows the format:
// 'embedded:filename.xslt'
func Path(name string) string {
	return fmt.Sprintf("embedded:%s", name)
}

// IsEmbeddedPath returns true if the path refers to an embedded XSLT file.
func IsEmbeddedPath(path string) bool {
	return len(path) > 9 && path[0:9] == "embedded:"
}

// ExtractNameFromPath extracts the filename from an embedded XSLT path.
func ExtractNameFromPath(path string) string {
	if IsEmbeddedPath(path) {
		return path[9:]
	}
	return path
}
