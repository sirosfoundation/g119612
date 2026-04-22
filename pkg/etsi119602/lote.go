// Package etsi119602 implements ETSI TS 119 602 Lists of Trusted Entities (LoTE).
//
// This package provides JSON-native types for representing trust lists in the
// EUDI wallet ecosystem. Unlike the XML-based TS 119 612, TS 119 602 uses JSON
// as its primary format and JWS (JSON Web Signature) for integrity protection.
//
// A LoTE contains:
//   - Scheme information identifying the list operator and list type
//   - A list of trusted entities with their roles, identifiers, and public keys
//   - Optional pointers to other LoTEs (for List-of-Lists patterns)
//   - An optional JWS signature
package etsi119602

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ListOfTrustedEntities is the root type for an ETSI TS 119 602 trust list.
type ListOfTrustedEntities struct {
	// Version identifies the LoTE schema version.
	Version string `json:"version"`

	// LOTETag is the XML LOTETag attribute value. When empty and marshalling to XML,
	// the default LOTETag constant is used.
	LOTETag string `json:"-"`

	// SchemeInformation describes the list operator and list metadata.
	SchemeInformation SchemeInformation `json:"schemeInformation"`

	// TrustedEntities is the list of trusted entities.
	TrustedEntities []TrustedEntity `json:"trustedEntities,omitempty"`

	// PointersToOtherLoTEs references other trust lists (for LoLoTE patterns).
	PointersToOtherLoTEs []LoTEPointer `json:"pointersToOtherLoTEs,omitempty"`
}

// SchemeInformation contains metadata about the trust list and its operator.
type SchemeInformation struct {
	// Territory is the country or region code (e.g. "EU", "SE", "DE").
	Territory string `json:"territory,omitempty"`

	// SchemeOperator identifies the operator of this trust list.
	SchemeOperator NameSet `json:"schemeOperator"`

	// SchemeName is the human-readable name of this trust list.
	SchemeName NameSet `json:"schemeName,omitempty"`

	// SchemeType is a URI identifying the type of list.
	SchemeType string `json:"schemeType"`

	// SchemeInformationURI points to human-readable information about the scheme.
	SchemeInformationURI []LangURI `json:"schemeInformationURI,omitempty"`

	// StatusDeterminationApproach is the URI of the status determination approach.
	StatusDeterminationApproach string `json:"statusDeterminationApproach,omitempty"`

	// PolicyOrLegalNotice contains legal or policy information.
	PolicyOrLegalNotice []LangString `json:"policyOrLegalNotice,omitempty"`

	// IssueDate is when this list was published.
	IssueDate time.Time `json:"issueDate"`

	// NextUpdate is when the next version is expected.
	NextUpdate *time.Time `json:"nextUpdate,omitempty"`

	// SequenceNumber orders list versions.
	SequenceNumber int `json:"sequenceNumber"`

	// DistributionPoints are URIs where this list can be fetched.
	DistributionPoints []string `json:"distributionPoints,omitempty"`
}

// TrustedEntity represents a single trusted entity in the list.
type TrustedEntity struct {
	// EntityID is the primary identifier for the entity (typically a URI).
	EntityID string `json:"entityId"`

	// EntityName is the human-readable name in one or more languages.
	EntityName NameSet `json:"entityName"`

	// EntityType classifies the entity (e.g. "credential-issuer", "verifier").
	EntityType string `json:"entityType,omitempty"`

	// EntityStatus indicates the current trust status.
	EntityStatus string `json:"entityStatus"`

	// StatusStartingTime is when the current status became effective.
	StatusStartingTime *time.Time `json:"statusStartingTime,omitempty"`

	// DigitalIdentities contains the entity's public keys and certificates.
	DigitalIdentities []DigitalIdentity `json:"digitalIdentities,omitempty"`

	// Services lists the trust services provided by this entity.
	Services []EntityService `json:"services,omitempty"`

	// InformationURIs point to additional information about the entity.
	InformationURIs []LangURI `json:"informationURIs,omitempty"`

	// Extensions allows scheme-specific additional data.
	Extensions map[string]any `json:"extensions,omitempty"`
}

// EntityService describes a specific trust service provided by an entity.
type EntityService struct {
	// ServiceType is a URI identifying the type of service.
	ServiceType string `json:"serviceType"`

	// ServiceName is the human-readable name.
	ServiceName NameSet `json:"serviceName,omitempty"`

	// ServiceStatus indicates the current status of this service.
	ServiceStatus string `json:"serviceStatus"`

	// StatusStartingTime is when the current status became effective.
	StatusStartingTime *time.Time `json:"statusStartingTime,omitempty"`

	// DigitalIdentities contains keys/certs specific to this service.
	DigitalIdentities []DigitalIdentity `json:"digitalIdentities,omitempty"`

	// ServiceSupplyPoints are endpoints where the service is available.
	ServiceSupplyPoints []string `json:"serviceSupplyPoints,omitempty"`

	// Extensions allows scheme-specific additional data.
	Extensions map[string]any `json:"extensions,omitempty"`
}

// DigitalIdentity represents a public key or certificate for an entity.
type DigitalIdentity struct {
	// Type indicates the identity format: "jwk", "x509", "did", "x509_subject_name".
	Type string `json:"type"`

	// JWK is a JSON Web Key (when Type == "jwk").
	JWK map[string]any `json:"jwk,omitempty"`

	// X509Certificate is a base64-encoded DER X.509 certificate (when Type == "x509").
	X509Certificate string `json:"x509Certificate,omitempty"`

	// X509SubjectName is the DN of the certificate subject (when Type == "x509_subject_name").
	X509SubjectName string `json:"x509SubjectName,omitempty"`

	// DID is a decentralized identifier (when Type == "did").
	DID string `json:"did,omitempty"`
}

// LoTEPointer references another List of Trusted Entities.
type LoTEPointer struct {
	// Location is the URI where the pointed-to LoTE can be fetched.
	Location string `json:"location"`

	// SchemeTerritory is the territory of the pointed-to list.
	SchemeTerritory string `json:"schemeTerritory,omitempty"`

	// SchemeType is the type URI of the pointed-to list.
	SchemeType string `json:"schemeType,omitempty"`

	// DigitalIdentities contains keys used to sign the pointed-to list.
	DigitalIdentities []DigitalIdentity `json:"digitalIdentities,omitempty"`

	// AdditionalInformation provides extra context about the pointer.
	AdditionalInformation map[string]any `json:"additionalInformation,omitempty"`
}

// --- Multi-language support types ---

// LangString is a string value with a language tag.
type LangString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// LangURI is a URI with a language tag.
type LangURI struct {
	Language string `json:"language"`
	URI      string `json:"uri"`
}

// NameSet is an ordered list of names in different languages.
type NameSet []LangString

// Get returns the name for the given language, or the fallback if not found.
func (ns NameSet) Get(lang, fallback string) string {
	for _, n := range ns {
		if n.Language == lang {
			return n.Value
		}
	}
	if len(ns) > 0 {
		return ns[0].Value
	}
	return fallback
}

// --- Status constants ---

const (
	// StatusGranted indicates an active, trusted entity.
	StatusGranted = "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"

	// StatusWithdrawn indicates a formerly trusted entity whose trust has been revoked.
	StatusWithdrawn = "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/withdrawn"

	// StatusRecognisedAtNationalLevel indicates recognition at national level.
	StatusRecognisedAtNationalLevel = "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/recognisedatnationallevel"

	// LoTEVersion is the current schema version.
	LoTEVersion = "1.0"
)

// --- Serialization ---

// ParseLoTE parses a LoTE from JSON bytes.
func ParseLoTE(data []byte) (*ListOfTrustedEntities, error) {
	var lote ListOfTrustedEntities
	if err := json.Unmarshal(data, &lote); err != nil {
		return nil, fmt.Errorf("failed to parse LoTE: %w", err)
	}
	return &lote, nil
}

// ParseLoTEFromFile loads and parses a LoTE from a JSON file.
func ParseLoTEFromFile(path string) (*ListOfTrustedEntities, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read LoTE file %s: %w", path, err)
	}
	return ParseLoTE(data)
}

// Marshal serializes the LoTE to JSON.
func (l *ListOfTrustedEntities) Marshal() ([]byte, error) {
	return json.Marshal(l)
}

// MarshalIndent serializes the LoTE to indented JSON.
func (l *ListOfTrustedEntities) MarshalIndent() ([]byte, error) {
	return json.MarshalIndent(l, "", "  ")
}
