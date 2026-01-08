package pipeline

import (
	"crypto/x509"
	"os"
	"testing"
	"text/template"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Helper function to create a pipeline with a logger
func createTestPipeline(pipes []Pipe) *Pipeline {
	return &Pipeline{
		Pipes:  pipes,
		Logger: logging.NewLogger(logging.DebugLevel),
	}
}

func TestPipeline_Process_Success(t *testing.T) {
	RegisterFunction("testfunc", func(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
		assert.Equal(t, []string{"foo", "bar"}, args)
		if ctx == nil {
			t.Fatal("ctx should not be nil")
		}
		ctx.EnsureTSLTrees()
		// Create a dummy tree and add it to the stack
		tree := &TSLTree{Root: &TSLNode{TSL: &etsi119612.TSL{}}}
		ctx.AddTSLTree(tree) // simulate adding a TSL tree
		return ctx, nil
	})
	yamlData := `
- testfunc:
    - foo
    - bar
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := createTestPipeline(pipes)
	ctx, err := pl.Process(&Context{})
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.Equal(t, 1, ctx.TSLTrees.Size())
}

func TestPipeline_Process_UnknownMethod(t *testing.T) {
	yamlData := `
- foo: []
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := createTestPipeline(pipes)
	ctx, err := pl.Process(&Context{})
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "unknown methodName")
}

func TestPipeline_Process_FuncError(t *testing.T) {
	RegisterFunction("failfunc", func(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
		return ctx, os.ErrPermission
	})
	yamlData := `
- failfunc:
    - foo
`
	var pipes []Pipe
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.NoError(t, err)
	pl := createTestPipeline(pipes)
	ctx, err := pl.Process(&Context{})
	assert.Error(t, err)
	assert.NotNil(t, ctx)
	assert.Contains(t, err.Error(), "failed")
}

// TestPipeline_SelectStep tests the select pipeline step with a local test TSL XML file.
func TestPipeline_SelectStep(t *testing.T) {
	// Render the XML template with the generated test certificate
	tmplBytes, err := os.ReadFile("./testdata/test-tsl.xml")
	assert.NoError(t, err)
	tmpl, err := template.New("tsl").Parse(string(tmplBytes))
	assert.NoError(t, err)
	tmpfile, err := os.CreateTemp("", "test-tsl-*.xml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = tmpl.Execute(tmpfile, map[string]string{"X509Certificate": TestCertBase64})
	assert.NoError(t, err)
	tmpfile.Close()
	yamlData := "- load: [\"" + tmpfile.Name() + "\"]\n- select: []\n"
	var pipes []Pipe
	assert.NoError(t, yaml.Unmarshal([]byte(yamlData), &pipes))
	pl := &Pipeline{
		Pipes:  pipes,
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx, err := pl.Process(&Context{})
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.CertPool)
	if ctx.CertPool != nil {
		opts := x509.VerifyOptions{
			Roots: ctx.CertPool,
		}
		_, err := TestCert.Verify(opts)
		assert.NoError(t, err, "testCert should verify against the CertPool")
	}
}

func TestNewPipeline(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "pipeline-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	yamlContent := "- echo: [\"foo\", \"bar\"]"
	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	pl, err := NewPipeline(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewPipeline failed: %v", err)
	}
	if len(pl.Pipes) != 1 || pl.Pipes[0].MethodName != "echo" {
		t.Errorf("Expected one echo step, got: %+v", pl.Pipes)
	}

	// Test error case: file does not exist
	_, err = NewPipeline("/nonexistent/file.yaml")
	if err == nil {
		t.Errorf("Expected error for nonexistent file, got nil")
	}
}

func TestNewPipeline_EdgeCases(t *testing.T) {
	t.Run("Invalid YAML syntax", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "invalid-pipeline-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		// Write invalid YAML
		_, err = tmpfile.Write([]byte("invalid: yaml: content: ["))
		require.NoError(t, err)
		tmpfile.Close()

		_, err = NewPipeline(tmpfile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse pipeline YAML")
	})

	t.Run("Empty pipeline file", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "empty-pipeline-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		// Write empty array
		_, err = tmpfile.Write([]byte("[]"))
		require.NoError(t, err)
		tmpfile.Close()

		pl, err := NewPipeline(tmpfile.Name())
		require.NoError(t, err)
		assert.NotNil(t, pl)
		assert.Len(t, pl.Pipes, 0)
		assert.NotNil(t, pl.Logger)
	})

	t.Run("Pipeline with logger always set", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "pipeline-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		_, err = tmpfile.Write([]byte("- echo: []"))
		require.NoError(t, err)
		tmpfile.Close()

		pl, err := NewPipeline(tmpfile.Name())
		require.NoError(t, err)
		assert.NotNil(t, pl.Logger, "Pipeline should always have a logger")
	})
}

func TestSelectCertPool_EdgeCases(t *testing.T) {
	// No TSLs
	ctx := &Context{TSLs: nil}
	_, err := selectCertPool(nil, ctx)
	if err == nil || err.Error() != "no TSLs loaded" {
		t.Errorf("Expected error for no TSLs, got: %v", err)
	}

	// TSLs with no matching policy
	tsl1 := generateTSL("Service1", "http://service-type1", []string{"cert1"})
	stack := utils.NewStack[*etsi119612.TSL]()
	stack.Push(tsl1)
	ctx = &Context{TSLs: stack}
	ctx, err = selectCertPool(nil, ctx, "nonexistent-policy")
	if err != nil {
		t.Errorf("Expected no error for no matching policy, got: %v", err)
	}
	if ctx.CertPool == nil {
		t.Errorf("Expected CertPool to be set for no matching policy")
	}

	// Multiple TSLs with different service types
	tsl2 := generateTSL("Service2", "http://service-type2", []string{"cert2"})
	stack = utils.NewStack[*etsi119612.TSL]()
	stack.Push(tsl1)
	stack.Push(tsl2)
	ctx = &Context{TSLs: stack}
	ctx, err = selectCertPool(nil, ctx)
	if err != nil {
		t.Errorf("Expected no error for multiple TSLs, got: %v", err)
	}
	if ctx.CertPool == nil {
		t.Errorf("Expected CertPool to be set for multiple TSLs")
	}
}

func TestSetFetchOptions(t *testing.T) {
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx := NewContext()

	// Test default values are set when the function is called with no args
	ctx, err := SetFetchOptions(pl, ctx)
	if err != nil {
		t.Errorf("Unexpected error for SetFetchOptions with no args: %v", err)
	}
	if ctx.TSLFetchOptions == nil {
		t.Fatalf("Expected TSLFetchOptions to be initialized, but it's nil")
	}

	// Test setting User-Agent
	ctx, err = SetFetchOptions(pl, ctx, "user-agent:TestAgent/1.0")
	if err != nil {
		t.Errorf("Unexpected error for setting user-agent: %v", err)
	}
	if ctx.TSLFetchOptions.UserAgent != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent to be 'TestAgent/1.0', got '%s'", ctx.TSLFetchOptions.UserAgent)
	}

	// Test setting timeout
	ctx, err = SetFetchOptions(pl, ctx, "timeout:45s")
	if err != nil {
		t.Errorf("Unexpected error for setting timeout: %v", err)
	}
	if ctx.TSLFetchOptions.Timeout != 45*time.Second {
		t.Errorf("Expected timeout to be 45s, got %v", ctx.TSLFetchOptions.Timeout)
	}

	// Test invalid timeout
	_, err = SetFetchOptions(pl, ctx, "timeout:invalid")
	if err == nil {
		t.Errorf("Expected error for invalid timeout, got nil")
	}
}

func TestLoadTSLWithOptions(t *testing.T) {
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx := NewContext()

	// Setup initial fetch options
	ctx, err := SetFetchOptions(pl, ctx, "user-agent:TestAgent/2.0", "timeout:15s")
	if err != nil {
		t.Fatalf("Failed to set fetch options: %v", err)
	}

	// Create a mock TSL file
	tmpfile, err := os.CreateTemp("", "test-tsl-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write minimal valid XML for a TSL
	content := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <SchemeOperatorName>
      <Name xml:lang="en">Test Operator</Name>
    </SchemeOperatorName>
  </SchemeInformation>
  <TrustServiceProviderList>
  </TrustServiceProviderList>
</TrustServiceStatusList>
`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Update fetch options before loading the TSL
	ctx, err = SetFetchOptions(pl, ctx, "user-agent:FileLoader/1.0")
	if err != nil {
		t.Fatalf("Failed to set updated fetch options: %v", err)
	}

	// Test loading the TSL with the updated options
	ctx, err = loadTSL(pl, ctx, tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load TSL: %v", err)
	}

	// Verify the TSL was loaded
	if ctx.TSLTrees == nil || ctx.TSLTrees.IsEmpty() {
		t.Error("Expected TSL to be loaded, but TSL stack is empty")
	}

	// Verify the User-Agent was updated correctly
	if ctx.TSLFetchOptions.UserAgent != "FileLoader/1.0" {
		t.Errorf("Expected User-Agent to be updated to 'FileLoader/1.0', got '%s'",
			ctx.TSLFetchOptions.UserAgent)
	}
}

func TestSetFetchOptionsAccept(t *testing.T) {
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx := NewContext()

	// Test setting Accept headers
	ctx, err := SetFetchOptions(pl, ctx, "accept:application/xml,text/xml")
	if err != nil {
		t.Fatalf("Failed to set fetch options: %v", err)
	}

	// Verify the Accept headers were set correctly
	expected := []string{"application/xml", "text/xml"}
	if len(ctx.TSLFetchOptions.AcceptHeaders) != len(expected) {
		t.Errorf("Expected %d Accept headers, got %d", len(expected), len(ctx.TSLFetchOptions.AcceptHeaders))
	}

	for i, v := range expected {
		if i >= len(ctx.TSLFetchOptions.AcceptHeaders) || ctx.TSLFetchOptions.AcceptHeaders[i] != v {
			t.Errorf("Accept header at position %d doesn't match. Expected '%s', got '%s'",
				i, v, ctx.TSLFetchOptions.AcceptHeaders[i])
		}
	}
}

func TestLoadTSL_Errors(t *testing.T) {
	ctx := NewContext()
	pl := createTestPipeline(nil)

	// Initialize TSLFetchOptions
	ctx, err := SetFetchOptions(pl, ctx)
	if err != nil {
		t.Fatalf("Failed to set fetch options: %v", err)
	}

	// Invalid file path
	_, err = loadTSL(pl, ctx, "file:///nonexistent/path.xml")
	if err == nil {
		t.Errorf("Expected error for invalid file path, got nil")
	}

	// Invalid URL (unsupported scheme)
	_, err = loadTSL(pl, ctx, "ftp://example.com/tsl.xml")
	if err == nil {
		t.Errorf("Expected error for unsupported URL scheme, got nil")
	}

	// Invalid XML file
	tmpfile, err := os.CreateTemp("", "badxml-*.xml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte("not xml at all")); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpfile.Close()
	_, err = loadTSL(pl, ctx, "file://"+tmpfile.Name())
	if err == nil {
		t.Errorf("Expected error for invalid XML, got nil")
	}
}

func TestPipe_UnmarshalYAML_Errors(t *testing.T) {
	// Not a mapping node
	var pipes []Pipe
	yamlData := "- not-a-map"
	err := yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Pipe must be a map")

	// Mapping node with wrong structure (not a sequence for args)
	yamlData = "- testfunc: foo"
	err = yaml.Unmarshal([]byte(yamlData), &pipes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Pipe arguments must be a sequence")
}

// TestSetFetchOptions_EdgeCases tests additional edge cases for SetFetchOptions
func TestSetFetchOptions_EdgeCases(t *testing.T) {
	t.Run("max-depth zero disables references", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		ctx, err := SetFetchOptions(pl, ctx, "max-depth:0")

		require.NoError(t, err)
		assert.Equal(t, 0, ctx.TSLFetchOptions.MaxDereferenceDepth)
	})

	t.Run("max-depth negative enables unlimited", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		ctx, err := SetFetchOptions(pl, ctx, "max-depth:-1")

		require.NoError(t, err)
		assert.Equal(t, -1, ctx.TSLFetchOptions.MaxDereferenceDepth)
	})

	t.Run("prefer-xml with various truthy values", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}

		testCases := []struct {
			value    string
			expected bool
		}{
			{"true", true},
			{"1", true},
			{"yes", true},
			{"false", false},
			{"0", false},
			{"no", false},
			{"", false},
		}

		for _, tc := range testCases {
			ctx := NewContext()
			ctx, err := SetFetchOptions(pl, ctx, "prefer-xml:"+tc.value)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, ctx.Data["prefer_xml_over_pdf"],
				"prefer-xml:%s should result in %v", tc.value, tc.expected)
		}
	})

	t.Run("filter-territory empty value", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		ctx, err := SetFetchOptions(pl, ctx, "filter-territory:")

		require.NoError(t, err)
		filters := ctx.Data["tsl_filters"].(map[string][]string)
		assert.NotContains(t, filters, "territory")
	})

	t.Run("filter-service-type empty value", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		ctx, err := SetFetchOptions(pl, ctx, "filter-service-type:")

		require.NoError(t, err)
		filters := ctx.Data["tsl_filters"].(map[string][]string)
		assert.NotContains(t, filters, "service-type")
	})

	t.Run("accept headers with whitespace trimming", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		ctx, err := SetFetchOptions(pl, ctx, "accept:  application/xml  , text/xml , application/pdf  ")

		require.NoError(t, err)
		require.Len(t, ctx.TSLFetchOptions.AcceptHeaders, 3)
		assert.Equal(t, "application/xml", ctx.TSLFetchOptions.AcceptHeaders[0])
		assert.Equal(t, "text/xml", ctx.TSLFetchOptions.AcceptHeaders[1])
		assert.Equal(t, "application/pdf", ctx.TSLFetchOptions.AcceptHeaders[2])
	})

	t.Run("empty accept resets to defaults", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()
		ctx.EnsureTSLFetchOptions()

		// Set custom headers
		ctx.TSLFetchOptions.AcceptHeaders = []string{"custom/type"}

		// Reset with empty value
		ctx, err := SetFetchOptions(pl, ctx, "accept:")

		require.NoError(t, err)
		// Should not be the custom value anymore
		assert.NotEqual(t, []string{"custom/type"}, ctx.TSLFetchOptions.AcceptHeaders)
	})

	t.Run("unknown option does not error", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		ctx, err := SetFetchOptions(pl, ctx, "unknown-option:some-value")

		require.NoError(t, err)
		assert.NotNil(t, ctx)
	})

	t.Run("mix of valid and unknown options", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		ctx, err := SetFetchOptions(pl, ctx,
			"user-agent:Test/1.0",
			"unknown:value",
			"timeout:30s",
		)

		require.NoError(t, err)
		assert.Equal(t, "Test/1.0", ctx.TSLFetchOptions.UserAgent)
		assert.Equal(t, 30*time.Second, ctx.TSLFetchOptions.Timeout)
	})
}
