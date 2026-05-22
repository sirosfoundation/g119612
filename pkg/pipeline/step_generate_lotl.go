package pipeline

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	ValidityDays   int               `yaml:"validityDays,omitempty"`
	Pointers       []LoTLPointerMeta `yaml:"pointers,omitempty"`
}

// LoTLPointerMeta represents a pointer entry in the LoTL YAML metadata.
type LoTLPointerMeta struct {
	Location            string          `yaml:"location"`
	SchemeTerritory     string          `yaml:"schemeTerritory,omitempty"`
	SchemeType          string          `yaml:"schemeType,omitempty"`
	SchemeOperatorNames []MultiLangName `yaml:"schemeOperatorNames,omitempty"`
	MimeType            string          `yaml:"mimeType,omitempty"`
	CertFiles           []string        `yaml:"certFiles,omitempty"`
}

// GenerateLoTL generates a LoTL (List of Trusted Lists) from a YAML metadata file.
//
// The directory structure is:
//
//	root/
//	  └── lotl.yaml           # LoTL scheme metadata with pointers
//
// If the directory contains scheme.yaml instead of lotl.yaml, a LoTE is
// generated via GenerateLoTE. This allows generate-lote and generate-lotl
// to be used interchangeably with auto-detection.
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

	// Auto-detect: if lotl.yaml doesn't exist but scheme.yaml does, delegate
	if _, err := os.Stat(filepath.Join(rootDir, "lotl.yaml")); os.IsNotExist(err) {
		if _, sErr := os.Stat(filepath.Join(rootDir, "scheme.yaml")); sErr == nil {
			return GenerateLoTE(pl, ctx, args...)
		}
	}

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

	now := time.Now().UTC()
	validityDays := meta.ValidityDays
	if validityDays <= 0 {
		validityDays = 180
	}
	if validityDays > 3650 {
		return nil, fmt.Errorf("validityDays too large: %d (max 3650)", validityDays)
	}
	nextUpdate := now.Add(time.Duration(validityDays) * 24 * time.Hour)
	lotl := &etsi119602.ListOfTrustedLists{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       meta.Territory,
			SchemeOperatorName:    multiLangToNameSet(meta.OperatorNames),
			SchemeName:            multiLangToNameSet(meta.SchemeName),
			LoTEType:              meta.SchemeType,
			LoTESequenceNumber:    meta.SequenceNumber,
			ListIssueDateTime:     now.Format(time.RFC3339),
			NextUpdate:            nextUpdate.Format(time.RFC3339),
		},
	}

	// Build pointers from metadata
	for _, pm := range meta.Pointers {
		mimeType := pm.MimeType
		if mimeType == "" {
			mimeType = "application/json"
		}

		pointer := etsi119602.OtherLoTEPointer{
			LoTELocation:              pm.Location,
			ServiceDigitalIdentities: []etsi119602.ServiceDigitalIdentity{},
			LoTEQualifiers: []etsi119602.LoTEQualifier{{
				SchemeTerritory:    pm.SchemeTerritory,
				LoTEType:           pm.SchemeType,
				SchemeOperatorName: multiLangToNameSet(pm.SchemeOperatorNames),
				MimeType:           mimeType,
			}},
		}

		// Load signer certificates for the pointed-to list
		for _, certFile := range pm.CertFiles {
			certPath := filepath.Clean(certFile)
			if filepath.IsAbs(certPath) {
				return nil, fmt.Errorf("absolute paths not allowed in certFiles: %s", certFile)
			}
			certPath = filepath.Join(rootDir, certPath)
			// Ensure resolved path stays within rootDir
			rel, err := filepath.Rel(rootDir, certPath)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("certFile path escapes root directory: %s", certFile)
			}
			derBytes, err := loadCertificateDER(certPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load pointer cert file %s: %w", certFile, err)
			}
			pointer.ServiceDigitalIdentities = append(pointer.ServiceDigitalIdentities, etsi119602.ServiceDigitalIdentity{
				X509Certificates: []etsi119602.PKIOb{{Val: base64.StdEncoding.EncodeToString(derBytes)}},
			})
		}

		lotl.ListAndSchemeInformation.PointersToOtherLoTE = append(lotl.ListAndSchemeInformation.PointersToOtherLoTE, pointer)
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Generated LoTL",
			logging.F("pointers", len(lotl.ListAndSchemeInformation.PointersToOtherLoTE)),
			logging.F("territory", meta.Territory))
	}

	ctx.EnsureLoTLs()
	ctx.LoTLs.Push(lotl)
	return ctx, nil
}

func init() {
	RegisterFunction("generate-lotl", GenerateLoTL)
}
