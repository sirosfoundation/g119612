package etsi119602

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXMLRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	nextUpdate := now.Add(24 * time.Hour)

	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			Territory: "SE",
			SchemeOperator: NameSet{
				{Language: "en", Value: "Swedish Authority"},
				{Language: "sv", Value: "Svensk Myndighet"},
			},
			SchemeName: NameSet{
				{Language: "en", Value: "Sweden PID Providers"},
			},
			SchemeType:                  LoTETypePIDProviders,
			StatusDeterminationApproach: "http://uri.etsi.org/TrstSvc/TrustedList/StatusDetn/appropriate",
			PolicyOrLegalNotice: []LangString{
				{Language: "en", Value: "This is a test notice"},
			},
			SchemeInformationURI: []LangURI{
				{Language: "en", URI: "https://example.com/info"},
			},
			IssueDate:          now,
			NextUpdate:         &nextUpdate,
			SequenceNumber:     7,
			DistributionPoints: []string{"https://example.com/lote-SE.json"},
		},
		TrustedEntities: []TrustedEntity{
			{
				EntityID:     "https://issuer.example.com",
				EntityName:   NameSet{{Language: "en", Value: "Example Issuer"}},
				EntityType:   "credential-issuer",
				EntityStatus: StatusGranted,
				DigitalIdentities: []DigitalIdentity{
					{Type: "x509", X509Certificate: "MIIB..."},
					{Type: "x509_subject_name", X509SubjectName: "CN=Example"},
					{Type: "did", DID: "did:web:example.com"},
					{Type: "jwk", JWK: map[string]any{"kty": "EC", "crv": "P-256"}},
				},
				Services: []EntityService{
					{
						ServiceType:   "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
						ServiceName:   NameSet{{Language: "en", Value: "Qualified CA"}},
						ServiceStatus: StatusGranted,
						ServiceSupplyPoints: []string{
							"https://ca.example.com",
						},
					},
				},
				InformationURIs: []LangURI{
					{Language: "en", URI: "https://issuer.example.com/info"},
				},
			},
		},
		PointersToOtherLoTEs: []LoTEPointer{
			{
				Location:        "https://example.com/lote-DE.json",
				SchemeTerritory: "DE",
				SchemeType:      LoTETypePIDProviders,
				DigitalIdentities: []DigitalIdentity{
					{Type: "x509", X509Certificate: "MIIC..."},
				},
			},
		},
	}

	// Marshal to XML
	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)
	require.NotEmpty(t, xmlData)

	// Verify XML uses XSD structure
	xmlStr := string(xmlData)
	assert.Contains(t, xmlStr, "ListOfTrustedEntities")
	assert.Contains(t, xmlStr, "ListAndSchemeInformation")
	assert.Contains(t, xmlStr, "TrustedEntitiesList")
	assert.Contains(t, xmlStr, "TrustedEntityInformation")
	assert.Contains(t, xmlStr, "TrustedEntityServices")
	assert.Contains(t, xmlStr, "Swedish Authority")
	assert.Contains(t, xmlStr, "credential-issuer")
	assert.Contains(t, xmlStr, "LOTETag")

	// Parse back
	parsed, err := ParseLoTEXML(xmlData)
	require.NoError(t, err)

	// Scheme information round-trips
	assert.Equal(t, LoTEVersion, parsed.Version)
	assert.Equal(t, "SE", parsed.SchemeInformation.Territory)
	assert.Equal(t, LoTETypePIDProviders, parsed.SchemeInformation.SchemeType)
	assert.Equal(t, 7, parsed.SchemeInformation.SequenceNumber)
	assert.Len(t, parsed.SchemeInformation.SchemeOperator, 2)
	assert.Equal(t, "Swedish Authority", parsed.SchemeInformation.SchemeOperator.Get("en", ""))
	assert.Len(t, parsed.SchemeInformation.SchemeName, 1)
	assert.Len(t, parsed.SchemeInformation.SchemeInformationURI, 1)
	assert.Len(t, parsed.SchemeInformation.PolicyOrLegalNotice, 1)
	assert.Len(t, parsed.SchemeInformation.DistributionPoints, 1)
	assert.NotNil(t, parsed.SchemeInformation.NextUpdate)
	assert.Equal(t, now, parsed.SchemeInformation.IssueDate)

	// Verify entities
	require.Len(t, parsed.TrustedEntities, 1)
	entity := parsed.TrustedEntities[0]

	// EntityID comes from the first TEInformationURI
	assert.Equal(t, "https://issuer.example.com", entity.EntityID)

	// Entity-level properties come from the first XSD service
	assert.Equal(t, StatusGranted, entity.EntityStatus)
	assert.Equal(t, "credential-issuer", entity.EntityType)

	// Digital identities round-trip through the XSD DigitalIdentityType
	assert.Len(t, entity.DigitalIdentities, 4)
	assert.Equal(t, "x509", entity.DigitalIdentities[0].Type)
	assert.Equal(t, "MIIB...", entity.DigitalIdentities[0].X509Certificate)
	assert.Equal(t, "x509_subject_name", entity.DigitalIdentities[1].Type)
	assert.Equal(t, "CN=Example", entity.DigitalIdentities[1].X509SubjectName)
	assert.Equal(t, "did", entity.DigitalIdentities[2].Type)
	assert.Equal(t, "did:web:example.com", entity.DigitalIdentities[2].DID)
	assert.Equal(t, "jwk", entity.DigitalIdentities[3].Type)
	assert.NotEmpty(t, entity.DigitalIdentities[3].JWK)

	// Services: the first XSD service carries entity-level props,
	// additional domain services are appended after it.
	require.Len(t, entity.Services, 1)
	assert.Equal(t, "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", entity.Services[0].ServiceType)
	assert.Len(t, entity.Services[0].ServiceSupplyPoints, 1)

	// InformationURIs (EntityID is excluded, rest preserved)
	assert.Len(t, entity.InformationURIs, 1)
	assert.Equal(t, "https://issuer.example.com/info", entity.InformationURIs[0].URI)

	// Verify pointers
	require.Len(t, parsed.PointersToOtherLoTEs, 1)
	ptr := parsed.PointersToOtherLoTEs[0]
	assert.Equal(t, "https://example.com/lote-DE.json", ptr.Location)
	assert.Equal(t, "DE", ptr.SchemeTerritory)
	assert.Equal(t, LoTETypePIDProviders, ptr.SchemeType)
	assert.Len(t, ptr.DigitalIdentities, 1)
}

func TestXMLLoTLRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	lotl := &ListOfTrustedLists{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			Territory: "EU",
			SchemeOperator: NameSet{
				{Language: "en", Value: "European Commission"},
			},
			SchemeType:     LoTLTypeEU,
			IssueDate:      now,
			SequenceNumber: 1,
		},
		PointersToOtherLoTEs: []LoTEPointer{
			{
				Location:        "https://example.com/lote-pid.json",
				SchemeTerritory: "EU",
				SchemeType:      LoTETypePIDProviders,
			},
		},
	}

	// Marshal to XML
	xmlData, err := lotl.EncodeXML()
	require.NoError(t, err)
	xmlStr := string(xmlData)
	assert.Contains(t, xmlStr, "ListAndSchemeInformation")
	assert.Contains(t, xmlStr, "European Commission")

	// Parse back as LoTL
	parsed, err := ParseLoTLXML(xmlData)
	require.NoError(t, err)

	assert.Equal(t, LoTEVersion, parsed.Version)
	assert.Equal(t, "EU", parsed.SchemeInformation.Territory)
	assert.Equal(t, LoTLTypeEU, parsed.SchemeInformation.SchemeType)
	assert.Len(t, parsed.PointersToOtherLoTEs, 1)
	assert.Equal(t, LoTETypePIDProviders, parsed.PointersToOtherLoTEs[0].SchemeType)
}

func TestXMLEmptyLoTE(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			SchemeOperator: NameSet{{Language: "en", Value: "Test"}},
			SchemeType:     "http://example.com/type",
			IssueDate:      now,
		},
	}

	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)

	parsed, err := ParseLoTEXML(xmlData)
	require.NoError(t, err)

	assert.Equal(t, LoTEVersion, parsed.Version)
	assert.Empty(t, parsed.TrustedEntities)
	assert.Empty(t, parsed.PointersToOtherLoTEs)
}

// TestParseIDUnionLoTE is an integration test that parses a real XML LoTE
// published by the IDunion trust registry (https://tl-api.dev.idunion.info).
// The test data was downloaded from:
//
//	https://tl-api.dev.idunion.info/api/v1/lDkk32bu/etsi/tl.xml
func TestParseIDUnionLoTE(t *testing.T) {
	const testFile = "testdata/idunion_lote.xml"
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Skipf("testdata not available: %v", err)
	}

	lote, err := ParseLoTEXML(data)
	require.NoError(t, err, "ParseLoTEXML should successfully parse the IDunion LoTE")

	// LOTETag should be preserved (IDunion uses empty string)
	assert.Empty(t, lote.LOTETag, "IDunion LoTE has empty LOTETag")

	// --- Scheme information ---
	si := lote.SchemeInformation
	assert.Equal(t, "http://uri.etsi.org/19602/LoTEType/EUWalletProvidersList", si.SchemeType)
	assert.Equal(t, "http://uri.etsi.org/19602/WalletProvidersList/StatusDetn/EU", si.StatusDeterminationApproach)
	assert.Equal(t, 1, si.SequenceNumber)

	// Scheme operator name (may have whitespace from XML pretty-printing)
	require.NotEmpty(t, si.SchemeOperator)
	operatorName := strings.TrimSpace(si.SchemeOperator.Get("en", ""))
	assert.Equal(t, "WEBUILD - WP 4 - Group 5 - Trust Registry Infrastructure", operatorName)

	// Scheme name
	require.NotEmpty(t, si.SchemeName)
	schemeName := strings.TrimSpace(si.SchemeName.Get("en", ""))
	assert.Equal(t, "EU:EN_Random Trusted List", schemeName)

	// Issue date should be valid
	assert.False(t, si.IssueDate.IsZero(), "IssueDate should be set")

	// NextUpdate should be set and after IssueDate
	require.NotNil(t, si.NextUpdate, "NextUpdate should be set")
	assert.True(t, si.NextUpdate.After(si.IssueDate), "NextUpdate should be after IssueDate")

	// LoTEPolicy should be parsed into PolicyOrLegalNotice
	require.NotEmpty(t, si.PolicyOrLegalNotice, "IDunion LoTE has a LoTEPolicy element")
	assert.Contains(t, si.PolicyOrLegalNotice[0].Value, "github.com/webuild-consortium")

	// --- Trusted entities ---
	require.GreaterOrEqual(t, len(lote.TrustedEntities), 1, "IDunion LoTE should have at least 1 trusted entity")

	// Check each entity has basic structure
	for i, e := range lote.TrustedEntities {
		assert.NotEmpty(t, e.EntityName, "entity %d should have a name", i)
		assert.NotEmpty(t, e.EntityType, "entity %d should have a type", i)
		assert.Equal(t, "http://uri.etsi.org/19602/SvcType/WalletSolution/Issuance", e.EntityType,
			"entity %d should have WalletSolution/Issuance type", i)
		assert.NotEmpty(t, e.EntityID, "entity %d should have an entity ID", i)
	}

	// Spot-check first entity ("org")
	e0 := lote.TrustedEntities[0]
	assert.Equal(t, "org", strings.TrimSpace(e0.EntityName.Get("en", "")))

	// Should have at least one X.509 certificate digital identity
	require.NotEmpty(t, e0.DigitalIdentities, "first entity should have digital identities")
	assert.Equal(t, "x509", e0.DigitalIdentities[0].Type)
	assert.NotEmpty(t, e0.DigitalIdentities[0].X509Certificate, "X.509 cert should not be empty")

	// Spot-check second entity ("ACME, inc") - if present
	if len(lote.TrustedEntities) >= 2 {
		e1 := lote.TrustedEntities[1]
		assert.Equal(t, "ACME, inc", strings.TrimSpace(e1.EntityName.Get("en", "")))
		require.NotEmpty(t, e1.DigitalIdentities)
		assert.Equal(t, "x509", e1.DigitalIdentities[0].Type)
	}

	// Verify round-trip: marshal parsed LoTE back to XML and re-parse
	xmlOut, err := lote.EncodeXML()
	require.NoError(t, err, "EncodeXML should succeed on parsed LoTE")
	require.NotEmpty(t, xmlOut)

	reparsed, err := ParseLoTEXML(xmlOut)
	require.NoError(t, err, "re-parsing marshalled XML should succeed")
	assert.Len(t, reparsed.TrustedEntities, len(lote.TrustedEntities),
		"round-tripped LoTE should preserve entity count")
	assert.Equal(t, si.SchemeType, reparsed.SchemeInformation.SchemeType)
	// LOTETag: IDunion has empty, round-trip fills default, reparsed preserves it
	assert.Equal(t, LOTETag, reparsed.LOTETag, "round-tripped LOTETag should be the default constant")
}

func TestParseLoTEXML_MalformedXML(t *testing.T) {
	_, err := ParseLoTEXML([]byte("not xml at all"))
	assert.Error(t, err, "should fail on non-XML input")
}

func TestParseLoTEXML_EmptyInput(t *testing.T) {
	_, err := ParseLoTEXML([]byte{})
	assert.Error(t, err, "should fail on empty input")
}

func TestParseLoTEXML_TruncatedXML(t *testing.T) {
	truncated := []byte(`<?xml version="1.0"?><ListOfTrustedEntitiesType LOTETag="">
		<ListAndSchemeInformation><LoTEVersionIdentifier>1</LoTEVersionIdentifier>`)
	_, err := ParseLoTEXML(truncated)
	assert.Error(t, err, "should fail on truncated XML")
}

func TestParseLoTEXML_MinimalValid(t *testing.T) {
	minimal := []byte(`<?xml version="1.0"?><ListOfTrustedEntitiesType LOTETag="test-tag">
		<ListAndSchemeInformation>
			<LoTEVersionIdentifier>1</LoTEVersionIdentifier>
			<LoTESequenceNumber>1</LoTESequenceNumber>
			<LoTEType>http://example.com/type</LoTEType>
			<StatusDeterminationApproach>http://example.com/status</StatusDeterminationApproach>
			<ListIssueDateTime>2026-01-01T00:00:00Z</ListIssueDateTime>
		</ListAndSchemeInformation>
	</ListOfTrustedEntitiesType>`)
	lote, err := ParseLoTEXML(minimal)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/type", lote.SchemeInformation.SchemeType)
	assert.Equal(t, "test-tag", lote.LOTETag)
}

func TestXMLPolicyRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("URI policy uses LoTEPolicy element", func(t *testing.T) {
		lote := &ListOfTrustedEntities{
			Version: LoTEVersion,
			SchemeInformation: SchemeInformation{
				SchemeOperator: NameSet{{Language: "en", Value: "Test"}},
				SchemeType:     "http://example.com/type",
				IssueDate:      now,
				PolicyOrLegalNotice: []LangString{
					{Language: "en", Value: "https://example.com/policy"},
				},
			},
		}

		xmlData, err := lote.EncodeXML()
		require.NoError(t, err)

		// Should use LoTEPolicy for URI values
		xmlStr := string(xmlData)
		assert.Contains(t, xmlStr, "LoTEPolicy")
		assert.NotContains(t, xmlStr, "LoTELegalNotice")

		parsed, err := ParseLoTEXML(xmlData)
		require.NoError(t, err)
		require.Len(t, parsed.SchemeInformation.PolicyOrLegalNotice, 1)
		assert.Equal(t, "https://example.com/policy", parsed.SchemeInformation.PolicyOrLegalNotice[0].Value)
	})

	t.Run("text policy uses LoTELegalNotice element", func(t *testing.T) {
		lote := &ListOfTrustedEntities{
			Version: LoTEVersion,
			SchemeInformation: SchemeInformation{
				SchemeOperator: NameSet{{Language: "en", Value: "Test"}},
				SchemeType:     "http://example.com/type",
				IssueDate:      now,
				PolicyOrLegalNotice: []LangString{
					{Language: "en", Value: "This is a legal notice"},
				},
			},
		}

		xmlData, err := lote.EncodeXML()
		require.NoError(t, err)

		// Should use LoTELegalNotice for text values
		xmlStr := string(xmlData)
		assert.Contains(t, xmlStr, "LoTELegalNotice")
		assert.NotContains(t, xmlStr, "LoTEPolicy")

		parsed, err := ParseLoTEXML(xmlData)
		require.NoError(t, err)
		require.Len(t, parsed.SchemeInformation.PolicyOrLegalNotice, 1)
		assert.Equal(t, "This is a legal notice", parsed.SchemeInformation.PolicyOrLegalNotice[0].Value)
	})

	t.Run("mixed policy and legal notice", func(t *testing.T) {
		lote := &ListOfTrustedEntities{
			Version: LoTEVersion,
			SchemeInformation: SchemeInformation{
				SchemeOperator: NameSet{{Language: "en", Value: "Test"}},
				SchemeType:     "http://example.com/type",
				IssueDate:      now,
				PolicyOrLegalNotice: []LangString{
					{Language: "en", Value: "https://example.com/policy"},
					{Language: "en", Value: "This is a legal notice"},
				},
			},
		}

		xmlData, err := lote.EncodeXML()
		require.NoError(t, err)

		parsed, err := ParseLoTEXML(xmlData)
		require.NoError(t, err)
		require.Len(t, parsed.SchemeInformation.PolicyOrLegalNotice, 2)
	})
}

func TestXMLLOTETagDefault(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			SchemeOperator: NameSet{{Language: "en", Value: "Test"}},
			SchemeType:     "http://example.com/type",
			IssueDate:      now,
		},
	}

	// LOTETag not set → should default to the constant
	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), LOTETag)

	// LOTETag set to custom value → should use it
	lote.LOTETag = "custom-tag"
	xmlData, err = lote.EncodeXML()
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), "custom-tag")
	assert.NotContains(t, string(xmlData), LOTETag)
}
