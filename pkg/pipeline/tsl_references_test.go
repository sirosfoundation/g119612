package pipeline

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

func TestLoadTSLWithReferences(t *testing.T) {
	pl := createTestPipeline(nil)
	ctx := NewContext()

	// Set up a TSL with references
	mainTSLFile, err := os.CreateTemp("", "main-tsl-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(mainTSLFile.Name())

	// Create a referenced TSL file
	referencedTSLFile, err := os.CreateTemp("", "referenced-tsl-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(referencedTSLFile.Name())

	// Write a simple TSL to the referenced file
	referencedContent := `<?xml version="1.0" encoding="UTF-8"?>
<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#" xmlns:xml="http://www.w3.org/XML/1998/namespace">
  <tsl:SchemeInformation>
    <tsl:SchemeOperatorName>
      <tsl:Name xml:lang="en">Referenced TSL</tsl:Name>
    </tsl:SchemeOperatorName>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList>
    <tsl:TrustServiceProvider>
      <tsl:TSPInformation>
        <tsl:TSPName>
          <tsl:Name xml:lang="en">Referenced Provider</tsl:Name>
        </tsl:TSPName>
      </tsl:TSPInformation>
    </tsl:TrustServiceProvider>
  </tsl:TrustServiceProviderList>
</tsl:TrustServiceStatusList>
`
	if _, err := referencedTSLFile.WriteString(referencedContent); err != nil {
		t.Fatalf("Failed to write to referenced TSL file: %v", err)
	}
	referencedTSLFile.Close()

	// Write a TSL with a pointer to the referenced TSL
	mainContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<tsl:TrustServiceStatusList xmlns:tsl="http://uri.etsi.org/02231/v2#" xmlns:xml="http://www.w3.org/XML/1998/namespace">
  <tsl:SchemeInformation>
    <tsl:SchemeOperatorName>
      <tsl:Name xml:lang="en">Main TSL</tsl:Name>
    </tsl:SchemeOperatorName>
    <tsl:PointersToOtherTSL>
      <tsl:OtherTSLPointer>
        <tsl:TSLLocation>%s</tsl:TSLLocation>
      </tsl:OtherTSLPointer>
    </tsl:PointersToOtherTSL>
  </tsl:SchemeInformation>
  <tsl:TrustServiceProviderList>
    <tsl:TrustServiceProvider>
      <tsl:TSPInformation>
        <tsl:TSPName>
          <tsl:Name xml:lang="en">Main Provider</tsl:Name>
        </tsl:TSPName>
      </tsl:TSPInformation>
    </tsl:TrustServiceProvider>
  </tsl:TrustServiceProviderList>
</tsl:TrustServiceStatusList>
`, "file://"+referencedTSLFile.Name())

	if _, err := mainTSLFile.WriteString(mainContent); err != nil {
		t.Fatalf("Failed to write to main TSL file: %v", err)
	}
	mainTSLFile.Close()

	// Test with default max-depth (should load both TSLs)
	// Set an explicit max-depth parameter to ensure references are followed
	ctx, err = SetFetchOptions(pl, ctx, "max-depth:3")
	if err != nil {
		t.Fatalf("Failed to set fetch options: %v", err)
	}

	// Test directly with FetchTSLWithReferencesAndOptions for comparison
	t.Logf("Calling FetchTSLWithReferencesAndOptions directly")
	directTSLs, err := etsi119612.FetchTSLWithReferencesAndOptions("file://"+mainTSLFile.Name(), *ctx.TSLFetchOptions)
	if err != nil {
		t.Logf("Direct fetch error: %v", err)
	} else {
		t.Logf("Direct fetch found %d TSLs", len(directTSLs))
		for i, tsl := range directTSLs {
			t.Logf("TSL[%d] Source: %s", i, tsl.Source)
		}
	}

	// Now use loadTSL
	t.Logf("Calling loadTSL")
	ctx, err = loadTSL(pl, ctx, mainTSLFile.Name())
	if err != nil {
		t.Fatalf("Failed to load TSL: %v", err)
	}

	// Print the main TSL content for debugging
	if ctx.TSLTrees != nil && ctx.TSLTrees.Size() > 0 {
		tree, ok := ctx.TSLTrees.Peek()
		if ok && tree != nil && tree.Root != nil && tree.Root.TSL != nil {
			tsl := tree.Root.TSL
			t.Logf("Main TSL operator name: %s", tsl.SchemeOperatorName())
			if tsl.StatusList.TslSchemeInformation != nil && tsl.StatusList.TslSchemeInformation.TslPointersToOtherTSL != nil {
				t.Logf("Main TSL has %d pointers to other TSLs", len(tsl.StatusList.TslSchemeInformation.TslPointersToOtherTSL.TslOtherTSLPointer))
				for i, ptr := range tsl.StatusList.TslSchemeInformation.TslPointersToOtherTSL.TslOtherTSLPointer {
					t.Logf("Pointer %d: %s", i, ptr.TSLLocation)
				}
			} else {
				t.Logf("Main TSL has no pointers to other TSLs")
			}
		}
	}

	// Verify both TSLs were loaded
	if ctx.TSLs == nil || ctx.TSLs.Size() != 2 {
		t.Errorf("Expected 2 TSLs to be loaded, but got %d", ctx.TSLs.Size())
	} else {
		t.Logf("Successfully loaded %d TSLs", ctx.TSLs.Size())
		// Check that the stack has both TSLs in the correct order
		tslSlice := ctx.TSLs.ToSlice()
		if len(tslSlice) == 2 {
			// First one should be the referenced TSL (because we push in reverse order)
			if !strings.Contains(tslSlice[0].SchemeOperatorName(), "Referenced TSL") {
				t.Errorf("Expected first TSL to be Referenced TSL, but got %s", tslSlice[0].SchemeOperatorName())
			} else {
				t.Logf("First TSL is correctly the referenced one: %s", tslSlice[0].SchemeOperatorName())
			}

			// Second one should be the main TSL
			if !strings.Contains(tslSlice[1].SchemeOperatorName(), "Main TSL") {
				t.Errorf("Expected second TSL to be Main TSL, but got %s", tslSlice[1].SchemeOperatorName())
			} else {
				t.Logf("Second TSL is correctly the main one: %s", tslSlice[1].SchemeOperatorName())
			}
		}
	} // Test with max-depth:0 (should load only the main TSL)
	ctx = NewContext()
	ctx, err = loadTSL(pl, ctx, mainTSLFile.Name(), "max-depth:0")
	if err != nil {
		t.Fatalf("Failed to load TSL with max-depth:0: %v", err)
	}

	// Verify only the main TSL was loaded
	if ctx.TSLs == nil || ctx.TSLs.Size() != 1 {
		t.Errorf("Expected 1 TSL to be loaded with max-depth:0, but got %d", ctx.TSLs.Size())
	}

	// Check the TSL on the stack is the main one
	tsl, ok := ctx.TSLs.Peek()
	if !ok {
		t.Fatalf("Failed to peek at TSL stack: stack is empty")
	}
	if tsl == nil {
		t.Fatalf("Failed to peek at TSL stack: got nil")
	}
	if !strings.Contains(tsl.SchemeOperatorName(), "Main TSL") {
		t.Errorf("Expected main TSL on stack, got: %s", tsl.SchemeOperatorName())
	}
}
