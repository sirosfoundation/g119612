// Package etsi119602 — this file defines the List of Trusted Lists (LoTL) type.
//
// A LoTL is a higher-level document that references multiple LoTEs and other LoTLs.
// It follows the same structure as a LoTE but its TrustedEntities list is typically
// empty — the LoTL instead contains PointersToOtherLoTEs referencing typed LoTEs
// (PID providers, wallet providers, etc.) and potentially other LoTLs.
//
// Per WP4 trust group conventions, a LoTL is distinguished from a LoTE by its
// SchemeType URI and by primarily using PointersToOtherLoTEs rather than
// TrustedEntities.
package etsi119602

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Well-known LoTE/LoTL scheme type URIs per WP4 trust group implementation profile.
const (
	// LoTLTypeEU is the scheme type for the EU List of Trusted Lists.
	LoTLTypeEU = "http://uri.etsi.org/19602/LoTLType/EUListOfTrustedLists"

	// LoTETypePIDProviders identifies a PID Providers list.
	LoTETypePIDProviders = "http://uri.etsi.org/19602/LoTEType/EUPIDProvidersList"

	// LoTETypeWalletProviders identifies a Wallet Providers list.
	LoTETypeWalletProviders = "http://uri.etsi.org/19602/LoTEType/EUWalletProvidersList"

	// LoTETypePubEAAProviders identifies a PuB-EAA Providers list.
	LoTETypePubEAAProviders = "http://uri.etsi.org/19602/LoTEType/EUPubEAAProvidersList"

	// LoTETypeWRPACProviders identifies a WRPAC Providers list.
	LoTETypeWRPACProviders = "http://uri.etsi.org/19602/LoTEType/EUWRPACProvidersList"

	// LoTETypeWRPRCProviders identifies a WRPRC Providers list.
	LoTETypeWRPRCProviders = "http://uri.etsi.org/19602/LoTEType/EUWRPRCProvidersList"

	// LoTETypeRegistrarsAndRegisters identifies a Registrars/Registers list.
	LoTETypeRegistrarsAndRegisters = "http://uri.etsi.org/19602/LoTEType/EURegistrarsAndRegistersList"
)

// ListOfTrustedLists is a LoTL document that references multiple LoTEs.
// Structurally it reuses ListOfTrustedEntities but semantically its purpose
// is to aggregate pointers to other lists rather than to list entities directly.
type ListOfTrustedLists struct {
	// Version identifies the LoTL schema version.
	Version string `json:"version"`

	// SchemeInformation describes the LoTL operator and metadata.
	SchemeInformation SchemeInformation `json:"schemeInformation"`

	// PointersToOtherLoTEs references the LoTEs and LoTLs aggregated by this LoTL.
	PointersToOtherLoTEs []LoTEPointer `json:"pointersToOtherLoTEs"`
}

// Validate checks the LoTL for structural correctness.
func (l *ListOfTrustedLists) Validate() error {
	if l.Version == "" {
		return fmt.Errorf("version is required")
	}
	if err := l.SchemeInformation.validate(); err != nil {
		return fmt.Errorf("schemeInformation: %w", err)
	}
	for i, ptr := range l.PointersToOtherLoTEs {
		if err := ptr.validate(); err != nil {
			return fmt.Errorf("pointersToOtherLoTEs[%d]: %w", i, err)
		}
	}
	return nil
}

// ParseLoTL parses a LoTL from JSON bytes.
func ParseLoTL(data []byte) (*ListOfTrustedLists, error) {
	var lotl ListOfTrustedLists
	if err := json.Unmarshal(data, &lotl); err != nil {
		return nil, fmt.Errorf("failed to parse LoTL: %w", err)
	}
	return &lotl, nil
}

// ParseLoTLFromFile loads and parses a LoTL from a JSON file.
func ParseLoTLFromFile(path string) (*ListOfTrustedLists, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read LoTL file %s: %w", path, err)
	}
	return ParseLoTL(data)
}

// Marshal serializes the LoTL to JSON.
func (l *ListOfTrustedLists) Marshal() ([]byte, error) {
	return json.Marshal(l)
}

// MarshalIndent serializes the LoTL to indented JSON.
func (l *ListOfTrustedLists) MarshalIndent() ([]byte, error) {
	return json.MarshalIndent(l, "", "  ")
}

// IsLoTLSchemeType returns true if the given scheme type URI identifies
// a List of Trusted Lists rather than a List of Trusted Entities.
// This is used for auto-classification when loading ETSI TS 119 602 documents.
func IsLoTLSchemeType(schemeType string) bool {
	return strings.Contains(schemeType, "/LoTLType/")
}

// ToLoTE converts the LoTL to a LoTE representation for format interoperability.
// The resulting LoTE has no TrustedEntities and only PointersToOtherLoTEs.
func (l *ListOfTrustedLists) ToLoTE() *ListOfTrustedEntities {
	return &ListOfTrustedEntities{
		Version:              l.Version,
		SchemeInformation:    l.SchemeInformation,
		PointersToOtherLoTEs: l.PointersToOtherLoTEs,
	}
}

// LoTLFromLoTE creates a LoTL from a LoTE that has only pointers (no entities).
func LoTLFromLoTE(lote *ListOfTrustedEntities) *ListOfTrustedLists {
	return &ListOfTrustedLists{
		Version:              lote.Version,
		SchemeInformation:    lote.SchemeInformation,
		PointersToOtherLoTEs: lote.PointersToOtherLoTEs,
	}
}
