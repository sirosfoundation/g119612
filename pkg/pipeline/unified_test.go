package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/utils"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name        string
		location    string
		contentType string
		data        []byte
		want        Format
	}{
		// Extension-based detection
		{name: "xml extension", location: "https://example.com/tsl.xml", want: FormatTSL},
		{name: "json extension", location: "https://example.com/lote.json", want: FormatLoTE},
		{name: "jws extension", location: "https://example.com/lote.jws", want: FormatLoTE},
		{name: "xml extension with query", location: "https://example.com/tsl.xml?v=1", want: FormatTSL},

		// Content-Type based detection
		{name: "application/xml", location: "https://example.com/tsl", contentType: "application/xml", want: FormatTSL},
		{name: "text/xml", location: "https://example.com/tsl", contentType: "text/xml; charset=utf-8", want: FormatTSL},
		{name: "application/json", location: "https://example.com/lote", contentType: "application/json", want: FormatLoTE},

		// Content probing
		{name: "xml content", location: "https://example.com/data", data: []byte("<?xml version=\"1.0\""), want: FormatTSL},
		{name: "xml content with bom", location: "https://example.com/data", data: []byte("\xef\xbb\xbf<?xml"), want: FormatTSL},
		{name: "json object content", location: "https://example.com/data", data: []byte(`{"version": "1.0"}`), want: FormatLoTE},
		{name: "json array content", location: "https://example.com/data", data: []byte(`[{"id": 1}]`), want: FormatLoTE},
		{name: "jws compact", location: "https://example.com/data", data: []byte("eyJhbGciOiJSUzI1NiJ9.eyJmb28iOiJiYXIifQ.sig"), want: FormatLoTE},

		// Unknown
		{name: "empty location", location: "", want: FormatUnknown},
		{name: "empty data", location: "https://example.com/data", data: []byte(""), want: FormatUnknown},
		{name: "binary data", location: "https://example.com/data", data: []byte{0x00, 0x01, 0x02}, want: FormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.location, tt.contentType, tt.data)
			if got != tt.want {
				t.Errorf("DetectFormat(%q, %q, %v) = %q, want %q", tt.location, tt.contentType, tt.data, got, tt.want)
			}
		})
	}
}

func TestUnifiedLoad(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()

	// Create a simple XML TSL file
	tslContent := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLVersionIdentifier>5</TSLVersionIdentifier>
		<TSLSequenceNumber>1</TSLSequenceNumber>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeOperatorName>
			<Name xml:lang="en">Test Operator</Name>
		</SchemeOperatorName>
		<SchemeTerritory>XX</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`
	tslPath := filepath.Join(tmpDir, "test.xml")
	if err := os.WriteFile(tslPath, []byte(tslContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a simple JSON LoTE file
	loteContent := `{
		"version": "1.0",
		"schemeInformation": {
			"territory": "XX",
			"schemeOperator": [{"language": "en", "value": "Test Operator"}],
			"schemeType": "test",
			"sequenceNumber": 1,
			"issueDate": "2024-01-01T00:00:00Z"
		},
		"trustedEntities": []
	}`
	lotePath := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(lotePath, []byte(loteContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		location  string
		wantTSL   bool
		wantLoTE  bool
		wantError bool
	}{
		{name: "load xml as tsl", location: tslPath, wantTSL: true},
		{name: "load json as lote", location: lotePath, wantLoTE: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			// Create pipeline with logger to avoid nil pointer in LoadTSL
			pl := &Pipeline{Logger: logging.NewLogger(logging.ErrorLevel)}

			ctx, err := Load(pl, ctx, tt.location)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantTSL && (ctx.TSLs == nil || ctx.TSLs.Size() == 0) {
				t.Error("expected TSL to be loaded")
			}
			if tt.wantLoTE && (ctx.LoTEs == nil || ctx.LoTEs.Size() == 0) {
				t.Error("expected LoTE to be loaded")
			}
		})
	}
}

func TestUnifiedPublish(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a context with both TSL and LoTE
	ctx := &Context{
		TSLs:  utils.NewStack[*etsi119612.TSL](),
		LoTEs: utils.NewStack[*etsi119602.ListOfTrustedEntities](),
	}

	// Add a minimal TSL
	ctx.TSLs.Push(&etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TSLVersionIdentifier: 5,
				TslSchemeTerritory:   "XX",
			},
		},
	})

	// Add a minimal LoTE
	ctx.LoTEs.Push(&etsi119602.ListOfTrustedEntities{
		Version: "1.0",
		SchemeInformation: etsi119602.SchemeInformation{
			Territory:      "XX",
			SchemeType:     "test",
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Test Operator"}},
			IssueDate:      time.Now().UTC(),
		},
	})

	pl := &Pipeline{Logger: logging.NewLogger(logging.ErrorLevel)}
	_, err := Publish(pl, ctx, tmpDir)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Check that files were created
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) == 0 {
		t.Error("expected files to be published")
	}
}

func TestUnifiedGenerate(t *testing.T) {
	// Test with LoTE structure
	t.Run("detect LoTE structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create entities directory (LoTE indicator)
		entitiesDir := filepath.Join(tmpDir, "entities")
		if err := os.MkdirAll(entitiesDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create minimal scheme.yaml
		schemeYAML := `operatorNames:
  - language: en
    value: Test Operator
schemeType: test
territory: XX
`
		if err := os.WriteFile(filepath.Join(tmpDir, "scheme.yaml"), []byte(schemeYAML), 0644); err != nil {
			t.Fatal(err)
		}

		ctx := &Context{}
		pl := &Pipeline{}

		ctx, err := Generate(pl, ctx, tmpDir)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if ctx.LoTEs == nil || ctx.LoTEs.Size() == 0 {
			t.Error("expected LoTE to be generated")
		}
	})

	// Test with ambiguous structure (both providers and entities)
	t.Run("reject ambiguous structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		if err := os.MkdirAll(filepath.Join(tmpDir, "providers"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(tmpDir, "entities"), 0755); err != nil {
			t.Fatal(err)
		}

		ctx := &Context{}
		pl := &Pipeline{}

		_, err := Generate(pl, ctx, tmpDir)
		if err == nil {
			t.Error("expected error for ambiguous structure")
		}
	})
}

func TestUnifiedMerge(t *testing.T) {
	t.Run("merge multiple LoTEs", func(t *testing.T) {
		ctx := &Context{
			LoTEs: utils.NewStack[*etsi119602.ListOfTrustedEntities](),
		}

		// Add two LoTEs
		ctx.LoTEs.Push(&etsi119602.ListOfTrustedEntities{
			Version: "1.0",
			SchemeInformation: etsi119602.SchemeInformation{
				Territory: "SE",
			},
			TrustedEntities: []etsi119602.TrustedEntity{
				{EntityID: "urn:entity:1"},
			},
		})
		ctx.LoTEs.Push(&etsi119602.ListOfTrustedEntities{
			Version: "1.0",
			SchemeInformation: etsi119602.SchemeInformation{
				Territory: "DE",
			},
			TrustedEntities: []etsi119602.TrustedEntity{
				{EntityID: "urn:entity:2"},
			},
		})

		pl := &Pipeline{}
		ctx, err := Merge(pl, ctx)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}

		if ctx.LoTEs.Size() != 1 {
			t.Errorf("expected 1 merged LoTE, got %d", ctx.LoTEs.Size())
		}

		merged, ok := ctx.LoTEs.Peek()
		if !ok {
			t.Fatal("expected to peek merged LoTE")
		}
		if len(merged.TrustedEntities) != 2 {
			t.Errorf("expected 2 entities in merged LoTE, got %d", len(merged.TrustedEntities))
		}
	})

	t.Run("single item no merge needed", func(t *testing.T) {
		ctx := &Context{
			LoTEs: utils.NewStack[*etsi119602.ListOfTrustedEntities](),
		}
		ctx.LoTEs.Push(&etsi119602.ListOfTrustedEntities{Version: "1.0"})

		pl := &Pipeline{}
		ctx, err := Merge(pl, ctx)
		if err != nil {
			t.Fatalf("Merge failed: %v", err)
		}

		if ctx.LoTEs.Size() != 1 {
			t.Errorf("expected 1 LoTE (unchanged), got %d", ctx.LoTEs.Size())
		}
	})
}
