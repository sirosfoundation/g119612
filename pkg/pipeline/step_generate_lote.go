package pipeline

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"gopkg.in/yaml.v3"
)

// LoTESchemeMetadata represents the YAML structure for LoTE scheme metadata.
type LoTESchemeMetadata struct {
	OperatorNames  []MultiLangName `yaml:"operatorNames"`
	SchemeName     []MultiLangName `yaml:"schemeName,omitempty"`
	SchemeType     string          `yaml:"schemeType"`
	Territory      string          `yaml:"territory,omitempty"`
	SequenceNumber int             `yaml:"sequenceNumber,omitempty"`
}

// LoTEEntityMetadata represents the YAML structure for a trusted entity.
type LoTEEntityMetadata struct {
	Names      []MultiLangName       `yaml:"names"`
	EntityID   string                `yaml:"entityId"`
	EntityType string                `yaml:"entityType,omitempty"`
	Status     string                `yaml:"status"`
	Services   []LoTEServiceMetadata `yaml:"services,omitempty"`
}

// LoTEServiceMetadata represents a service entry for a LoTE entity.
type LoTEServiceMetadata struct {
	ServiceNames []MultiLangName `yaml:"serviceNames"`
	ServiceType  string          `yaml:"serviceType"`
	Status       string          `yaml:"status"`
}

// GenerateLoTE generates a LoTE (List of Trusted Entities) JSON document from a
// directory structure:
//
//	root/
//	  ├── scheme.yaml           # LoTE scheme metadata
//	  └── entities/             # One subdirectory per trusted entity
//	      └── entity1/
//	          ├── entity.yaml   # Entity metadata
//	          ├── cert1.pem     # X.509 certificate (DER in PEM-armored or raw DER)
//	          └── key1.jwk      # JWK key (JSON file)
//
// The generated LoTE is pushed onto ctx.LoTEs.
//
// If the directory contains lotl.yaml instead of scheme.yaml, a LoTL is
// generated and pushed onto ctx.LoTLs. This allows generate-lote and
// generate-lotl to be used interchangeably with auto-detection.
func GenerateLoTE(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("generate-lote requires 1 argument: path to root directory")
	}

	rootDir := args[0]

	// Auto-detect: if lotl.yaml exists, delegate to GenerateLoTL
	if _, err := os.Stat(filepath.Join(rootDir, "lotl.yaml")); err == nil {
		return GenerateLoTL(pl, ctx, args...)
	}

	// Load scheme metadata
	schemeData, err := os.ReadFile(filepath.Join(rootDir, "scheme.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read scheme.yaml in %s: %w", rootDir, err)
	}
	var scheme LoTESchemeMetadata
	if err := yaml.Unmarshal(schemeData, &scheme); err != nil {
		return nil, fmt.Errorf("failed to parse scheme.yaml: %w", err)
	}
	if len(scheme.OperatorNames) == 0 {
		return nil, fmt.Errorf("scheme.yaml must have at least one operatorName")
	}
	if scheme.SchemeType == "" {
		return nil, fmt.Errorf("scheme.yaml must have a schemeType")
	}

	lote := &etsi119602.ListOfTrustedEntities{
		Version: etsi119602.LoTEVersion,
		SchemeInformation: etsi119602.SchemeInformation{
			Territory:      scheme.Territory,
			SchemeOperator: multiLangToNameSet(scheme.OperatorNames),
			SchemeName:     multiLangToNameSet(scheme.SchemeName),
			SchemeType:     scheme.SchemeType,
			SequenceNumber: scheme.SequenceNumber,
			IssueDate:      time.Now().UTC(),
		},
	}

	// Load entities
	entitiesDir := filepath.Join(rootDir, "entities")
	entries, err := os.ReadDir(entitiesDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No entities directory is valid — empty list
			ctx.EnsureLoTEs()
			ctx.LoTEs.Push(lote)
			return ctx, nil
		}
		return nil, fmt.Errorf("failed to read entities directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entityDir := filepath.Join(entitiesDir, entry.Name())
		entity, err := loadLoTEEntity(entityDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load entity %s: %w", entry.Name(), err)
		}
		lote.TrustedEntities = append(lote.TrustedEntities, *entity)
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Generated LoTE",
			logging.F("entities", len(lote.TrustedEntities)),
			logging.F("territory", scheme.Territory))
	}

	ctx.EnsureLoTEs()
	ctx.LoTEs.Push(lote)
	return ctx, nil
}

func loadLoTEEntity(entityDir string) (*etsi119602.TrustedEntity, error) {
	// Load entity metadata
	metaData, err := os.ReadFile(filepath.Join(entityDir, "entity.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read entity.yaml: %w", err)
	}
	var meta LoTEEntityMetadata
	if err := yaml.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse entity.yaml: %w", err)
	}
	if len(meta.Names) == 0 {
		return nil, fmt.Errorf("entity.yaml must have at least one name")
	}
	if meta.Status == "" {
		meta.Status = etsi119602.StatusGranted
	}

	entity := &etsi119602.TrustedEntity{
		EntityID:     meta.EntityID,
		EntityName:   multiLangToNameSet(meta.Names),
		EntityType:   meta.EntityType,
		EntityStatus: meta.Status,
	}

	// Load digital identities from certificate and JWK files
	files, err := os.ReadDir(entityDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read entity directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		path := filepath.Join(entityDir, name)

		switch {
		case strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".crt") || strings.HasSuffix(name, ".cer"):
			certData, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read certificate %s: %w", name, err)
			}
			cert, err := parseCertificateFile(certData)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate %s: %w", name, err)
			}
			entity.DigitalIdentities = append(entity.DigitalIdentities, etsi119602.DigitalIdentity{
				Type:            "x509",
				X509Certificate: base64.StdEncoding.EncodeToString(cert.Raw),
			})

		case strings.HasSuffix(name, ".jwk"):
			jwkData, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read JWK %s: %w", name, err)
			}
			var jwk map[string]any
			if err := json.Unmarshal(jwkData, &jwk); err != nil {
				// Fall back to YAML parsing for YAML-formatted JWK files
				if err2 := yaml.Unmarshal(jwkData, &jwk); err2 != nil {
					return nil, fmt.Errorf("failed to parse JWK %s: %w", name, err)
				}
			}
			entity.DigitalIdentities = append(entity.DigitalIdentities, etsi119602.DigitalIdentity{
				Type: "jwk",
				JWK:  jwk,
			})

		case strings.HasSuffix(name, ".did"):
			didData, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read DID %s: %w", name, err)
			}
			did := strings.TrimSpace(string(didData))
			if !strings.HasPrefix(did, "did:") {
				return nil, fmt.Errorf("invalid DID in %s: must start with 'did:'", name)
			}
			entity.DigitalIdentities = append(entity.DigitalIdentities, etsi119602.DigitalIdentity{
				Type: "did",
				DID:  did,
			})
		}
	}

	// Convert service metadata
	for _, svc := range meta.Services {
		entity.Services = append(entity.Services, etsi119602.EntityService{
			ServiceType:   svc.ServiceType,
			ServiceName:   multiLangToNameSet(svc.ServiceNames),
			ServiceStatus: svc.Status,
		})
	}

	return entity, nil
}

func multiLangToNameSet(names []MultiLangName) etsi119602.NameSet {
	if len(names) == 0 {
		return nil
	}
	ns := make(etsi119602.NameSet, len(names))
	for i, n := range names {
		ns[i] = etsi119602.LangString{Language: n.Language, Value: n.Value}
	}
	return ns
}

// parseCertificateFile tries to parse certificate data as raw DER first,
// then falls back to PEM decoding.
func parseCertificateFile(data []byte) (*x509.Certificate, error) {
	// Try raw DER first
	cert, err := x509.ParseCertificate(data)
	if err == nil {
		return cert, nil
	}

	// Fall back to PEM decoding
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("not valid DER or PEM certificate data")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("PEM block type is %q, expected CERTIFICATE", block.Type)
	}
	return x509.ParseCertificate(block.Bytes)
}
