package pipeline

import (
	"fmt"
	"time"

	"os"
	"path/filepath"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"gopkg.in/yaml.v3"
)

// LoTLSchemeMetadata represents the YAML structure for LoTL scheme metadata.
type LoTLSchemeMetadata struct {
	OperatorNames  []MultiLangName   `yaml:"operatorNames"`
	SchemeName     []MultiLangName   `yaml:"schemeName,omitempty"`
	SchemeType     string            `yaml:"schemeType"`
	Territory      string            `yaml:"territory,omitempty"`
	SequenceNumber int               `yaml:"sequenceNumber,omitempty"`
	Pointers       []LoTLPointerMeta `yaml:"pointers,omitempty"`
}

// LoTLPointerMeta represents a pointer entry in the LoTL YAML metadata.
type LoTLPointerMeta struct {
	Location        string `yaml:"location"`
	SchemeTerritory string `yaml:"schemeTerritory,omitempty"`
	SchemeType      string `yaml:"schemeType,omitempty"`
}

// GenerateLoTL generates a LoTL (List of Trusted Lists) from a YAML metadata file.
//
// The directory structure is:
//
//	root/
//	  └── lotl.yaml           # LoTL scheme metadata with pointers
//
// Usage in pipeline YAML:
//
//   - generate-lotl:
//   - /path/to/root/dir
func GenerateLoTL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("generate-lotl requires 1 argument: path to root directory")
	}

	rootDir := args[0]

	// Load LoTL metadata
	metaData, err := os.ReadFile(filepath.Join(rootDir, "lotl.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read lotl.yaml in %s: %w", rootDir, err)
	}
	var meta LoTLSchemeMetadata
	if err := yaml.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse lotl.yaml: %w", err)
	}
	if len(meta.OperatorNames) == 0 {
		return nil, fmt.Errorf("lotl.yaml must have at least one operatorName")
	}
	if meta.SchemeType == "" {
		return nil, fmt.Errorf("lotl.yaml must have a schemeType")
	}

	lotl := &etsi119602.ListOfTrustedLists{
		Version: etsi119602.LoTEVersion,
		SchemeInformation: etsi119602.SchemeInformation{
			Territory:      meta.Territory,
			SchemeOperator: multiLangToNameSet(meta.OperatorNames),
			SchemeName:     multiLangToNameSet(meta.SchemeName),
			SchemeType:     meta.SchemeType,
			SequenceNumber: meta.SequenceNumber,
			IssueDate:      time.Now().UTC(),
		},
	}

	// Build pointers from metadata
	for _, pm := range meta.Pointers {
		lotl.PointersToOtherLoTEs = append(lotl.PointersToOtherLoTEs, etsi119602.LoTEPointer{
			Location:        pm.Location,
			SchemeTerritory: pm.SchemeTerritory,
			SchemeType:      pm.SchemeType,
		})
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Generated LoTL",
			logging.F("pointers", len(lotl.PointersToOtherLoTEs)),
			logging.F("territory", meta.Territory))
	}

	ctx.EnsureLoTLs()
	ctx.LoTLs.Push(lotl)
	return ctx, nil
}

func init() {
	RegisterFunction("generate-lotl", GenerateLoTL)
}
