package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestXSLTCache(t *testing.T) {
	// Clear cache before tests
	globalXSLTCache.clear()

	t.Run("Cache_Miss_And_Hit", func(t *testing.T) {
		loadCount := 0
		loader := func() ([]byte, error) {
			loadCount++
			return []byte("test content"), nil
		}

		// First call should load
		content1, err := globalXSLTCache.get("test-key", loader)
		if err != nil {
			t.Fatalf("First get failed: %v", err)
		}
		if string(content1) != "test content" {
			t.Errorf("Expected 'test content', got '%s'", string(content1))
		}
		if loadCount != 1 {
			t.Errorf("Expected loader to be called once, but was called %d times", loadCount)
		}

		// Second call should hit cache (loader not called)
		content2, err := globalXSLTCache.get("test-key", loader)
		if err != nil {
			t.Fatalf("Second get failed: %v", err)
		}
		if string(content2) != "test content" {
			t.Errorf("Expected 'test content', got '%s'", string(content2))
		}
		if loadCount != 1 {
			t.Errorf("Expected loader to still be called once, but was called %d times", loadCount)
		}
	})

	t.Run("Cache_Different_Keys", func(t *testing.T) {
		globalXSLTCache.clear()

		content1, err := globalXSLTCache.get("key1", func() ([]byte, error) {
			return []byte("content1"), nil
		})
		if err != nil {
			t.Fatalf("Get key1 failed: %v", err)
		}

		content2, err := globalXSLTCache.get("key2", func() ([]byte, error) {
			return []byte("content2"), nil
		})
		if err != nil {
			t.Fatalf("Get key2 failed: %v", err)
		}

		if string(content1) != "content1" {
			t.Errorf("Expected 'content1', got '%s'", string(content1))
		}
		if string(content2) != "content2" {
			t.Errorf("Expected 'content2', got '%s'", string(content2))
		}
	})

	t.Run("Cache_Loader_Error", func(t *testing.T) {
		globalXSLTCache.clear()

		expectedErr := fmt.Errorf("load error")
		_, err := globalXSLTCache.get("error-key", func() ([]byte, error) {
			return nil, expectedErr
		})

		if err == nil {
			t.Fatal("Expected error from loader, got nil")
		}
		if err != expectedErr {
			t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
		}
	})

	t.Run("Cache_Clear", func(t *testing.T) {
		globalXSLTCache.clear()

		loadCount := 0
		loader := func() ([]byte, error) {
			loadCount++
			return []byte("test content"), nil
		}

		// Load into cache
		_, err := globalXSLTCache.get("clear-test", loader)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if loadCount != 1 {
			t.Errorf("Expected loader called once, got %d", loadCount)
		}

		// Clear cache
		globalXSLTCache.clear()

		// Should load again
		_, err = globalXSLTCache.get("clear-test", loader)
		if err != nil {
			t.Fatalf("Get after clear failed: %v", err)
		}
		if loadCount != 2 {
			t.Errorf("Expected loader called twice after clear, got %d", loadCount)
		}
	})
}

func TestFileXSLTCaching(t *testing.T) {
	// Clear cache before test
	globalXSLTCache.clear()

	// Create a temporary XSLT file
	tempDir := t.TempDir()
	xsltPath := filepath.Join(tempDir, "test.xslt")
	xsltContent := []byte(`<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" indent="yes"/>
  <xsl:template match="/">
    <result>transformed</result>
  </xsl:template>
</xsl:stylesheet>`)

	if err := os.WriteFile(xsltPath, xsltContent, 0644); err != nil {
		t.Fatalf("Failed to create test XSLT file: %v", err)
	}

	xmlData := []byte(`<?xml version="1.0"?><input>test</input>`)

	// First transformation - should cache the XSLT
	result1, err := applyFileXSLTTransformation(xmlData, xsltPath)
	if err != nil {
		t.Fatalf("First transformation failed: %v", err)
	}

	// Verify cache was populated
	cacheKey := "file:" + xsltPath
	globalXSLTCache.mu.RLock()
	_, inCache := globalXSLTCache.cache[cacheKey]
	globalXSLTCache.mu.RUnlock()

	if !inCache {
		t.Error("Expected XSLT to be cached after first transformation")
	}

	// Second transformation - should use cache
	result2, err := applyFileXSLTTransformation(xmlData, xsltPath)
	if err != nil {
		t.Fatalf("Second transformation failed: %v", err)
	}

	// Results should be identical
	if string(result1) != string(result2) {
		t.Errorf("Results differ:\nFirst: %s\nSecond: %s", string(result1), string(result2))
	}

	// Verify the transformation worked
	if !containsSubstring(result1, "transformed") {
		t.Errorf("Expected transformation output to contain 'transformed', got: %s", string(result1))
	}
}

func TestEmbeddedXSLTCaching(t *testing.T) {
	// Clear cache before test
	globalXSLTCache.clear()

	// Use a known embedded XSLT (tsl-to-html.xslt should be available)
	xsltName := "tsl-to-html.xslt"
	xmlData := []byte(`<?xml version="1.0"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLSequenceNumber>1</TSLSequenceNumber>
  </SchemeInformation>
</TrustServiceStatusList>`)

	// First transformation - should cache the XSLT
	result1, err := applyEmbeddedXSLTTransformation(xmlData, xsltName)
	if err != nil {
		t.Fatalf("First transformation failed: %v", err)
	}

	// Verify cache was populated
	cacheKey := "embedded:" + xsltName
	globalXSLTCache.mu.RLock()
	_, inCache := globalXSLTCache.cache[cacheKey]
	globalXSLTCache.mu.RUnlock()

	if !inCache {
		t.Error("Expected embedded XSLT to be cached after first transformation")
	}

	// Second transformation - should use cache
	result2, err := applyEmbeddedXSLTTransformation(xmlData, xsltName)
	if err != nil {
		t.Fatalf("Second transformation failed: %v", err)
	}

	// Results should be identical
	if string(result1) != string(result2) {
		t.Errorf("Results differ:\nFirst: %s\nSecond: %s", string(result1), string(result2))
	}

	// Verify the transformation produced HTML
	if !containsSubstring(result1, "html") {
		t.Errorf("Expected HTML output from tsl-to-html.xslt transformation")
	}
}

// Helper function to check if byte slice contains substring
func containsSubstring(data []byte, substr string) bool {
	return len(data) > 0 && len(substr) > 0 &&
		len(string(data)) >= len(substr) &&
		string(data) != "" &&
		substr != "" &&
		findSubstring(string(data), substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
