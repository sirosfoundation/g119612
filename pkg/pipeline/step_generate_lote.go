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
	Names          []MultiLangName       `yaml:"names"`
	EntityID       string                `yaml:"entityId"`
	EntityType     string                `yaml:"entityType,omitempty"`
	Status         string                `yaml:"status"`
	Address        *Address              `yaml:"address,omitempty"`
	InformationURI []MultiLangName       `yaml:"informationURI,omitempty"`
	Services       []LoTEServiceMetadata `yaml:"services,omitempty"`
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

	now := time.Now().UTC().Format(time.RFC3339)
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       scheme.Territory,
			SchemeOperatorName:    multiLangToNameSet(scheme.OperatorNames),
			SchemeName:            multiLangToNameSet(scheme.SchemeName),
			LoTEType:              scheme.SchemeType,
			LoTESequenceNumber:    scheme.SequenceNumber,
			ListIssueDateTime:     now,
			NextUpdate:            now,
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
		entity, err := loadLoTEEntity(entityDir, scheme.SchemeType)
		if err != nil {
			return nil, fmt.Errorf("failed to load entity %s: %w", entry.Name(), err)
		}
		lote.TrustedEntitiesList = append(lote.TrustedEntitiesList, *entity)
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Generated LoTE",
			logging.F("entities", len(lote.TrustedEntitiesList)),
			logging.F("territory", scheme.Territory))
	}

	ctx.EnsureLoTEs()
	ctx.LoTEs.Push(lote)
	return ctx, nil
}

func loadLoTEEntity(entityDir string, schemeType string) (*etsi119602.TrustedEntity, error) {
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

	// Per ETSI TS 119 602 Annexes D–I: only Pub-EAA uses explicit ServiceStatus.
	// For all other profiles, presence in the list = trusted (ServiceStatus forbidden).
	useServiceStatus := etsi119602.IsPubEAASchemeType(schemeType)
	if meta.Status == "" {
		meta.Status = etsi119602.StatusGranted
	}

	entity := &etsi119602.TrustedEntity{
		TrustedEntityInformation: etsi119602.TrustedEntityInformation{
			TEName:           multiLangToNameSet(meta.Names),
			TEAddress:        addressToTEAddress(meta.Address),
			TEInformationURI: multiLangToURIs(meta.InformationURI),
		},
	}

	// Collect digital identities from certificate, JWK, and DID files
	var sdi etsi119602.ServiceDigitalIdentity
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
			sdi.X509Certificates = append(sdi.X509Certificates,
				etsi119602.PKIOb{Val: base64.StdEncoding.EncodeToString(cert.Raw)})

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
			sdi.PublicKeyValues = append(sdi.PublicKeyValues, jwk)

		case strings.HasSuffix(name, ".did"):
			didData, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read DID %s: %w", name, err)
			}
			did := strings.TrimSpace(string(didData))
			if !strings.HasPrefix(did, "did:") {
				return nil, fmt.Errorf("invalid DID in %s: must start with 'did:'", name)
			}
			sdi.OtherIds = append(sdi.OtherIds, did)
		}
	}

	// Convert service metadata
	for _, svc := range meta.Services {
		si := etsi119602.ServiceInformation{
			ServiceTypeIdentifier:  svc.ServiceType,
			ServiceName:            multiLangToNameSet(svc.ServiceNames),
			ServiceDigitalIdentity: sdi,
		}
		if useServiceStatus {
			status := svc.Status
			if status == "" {
				status = meta.Status
			}
			si.ServiceStatus = status
		}
		entity.TrustedEntityServices = append(entity.TrustedEntityServices, etsi119602.TrustedEntityService{
			ServiceInformation: si,
		})
	}

	// If no services defined, create a default service from the entity-level identities
	if len(entity.TrustedEntityServices) == 0 {
		si := etsi119602.ServiceInformation{
			ServiceName:            multiLangToNameSet(meta.Names),
			ServiceDigitalIdentity: sdi,
		}
		if useServiceStatus {
			si.ServiceStatus = meta.Status
		}
		entity.TrustedEntityServices = []etsi119602.TrustedEntityService{{
			ServiceInformation: si,
		}}
	}

	return entity, nil
}

func multiLangToNameSet(names []MultiLangName) etsi119602.NameSet {
	if len(names) == 0 {
		return nil
	}
	ns := make(etsi119602.NameSet, len(names))
	for i, n := range names {
		ns[i] = etsi119602.MultiLangString{Lang: n.Language, Value: n.Value}
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

// addressToTEAddress converts a YAML Address to an ETSI TEAddress.
func addressToTEAddress(addr *Address) *etsi119602.TEAddress {
	if addr == nil {
		return nil
	}
	teAddr := &etsi119602.TEAddress{
		TEPostalAddress: []etsi119602.PostalAddress{{
			StreetAddress:   addr.Postal.StreetAddress,
			Locality:        addr.Postal.Locality,
			StateOrProvince: addr.Postal.StateOrProvince,
			PostalCode:      addr.Postal.PostalCode,
			Country:         addr.Postal.CountryName,
		}},
	}
	for _, e := range addr.Electronic {
		teAddr.TEElectronicAddress = append(teAddr.TEElectronicAddress,
			etsi119602.NonEmptyMultiLangURI{URIValue: e})
	}
	return teAddr
}

// multiLangToURIs converts multi-language names to NonEmptyMultiLangURI slice.
func multiLangToURIs(names []MultiLangName) []etsi119602.NonEmptyMultiLangURI {
	if len(names) == 0 {
		return nil
	}
	uris := make([]etsi119602.NonEmptyMultiLangURI, len(names))
	for i, n := range names {
		uris[i] = etsi119602.NonEmptyMultiLangURI{Lang: n.Language, URIValue: n.Value}
	}
	return uris
}
