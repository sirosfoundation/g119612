package etsi119602

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLoTE() *ListOfTrustedEntities {
	return &ListOfTrustedEntities{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    42,
			LoTEType:              LoTETypePIDProviders,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "Test Operator"}},
			SchemeTerritory:       "SE",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
		},
		TrustedEntitiesList: []TrustedEntity{
			{
				TrustedEntityInformation: TrustedEntityInformation{
					TEName:           NameSet{{Lang: "en", Value: "Example Issuer"}},
					TEInformationURI: []NonEmptyMultiLangURI{{Lang: "en", URIValue: "https://issuer.example.com"}},
				},
				TrustedEntityServices: []TrustedEntityService{{
					ServiceInformation: ServiceInformation{
						ServiceName: NameSet{{Lang: "en", Value: "Example Issuer"}},
						ServiceDigitalIdentity: ServiceDigitalIdentity{
							PublicKeyValues: []map[string]any{
								{"kty": "EC", "crv": "P-256", "x": "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU", "y": "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0"},
							},
						},
						ServiceTypeIdentifier: "credential-issuer",
						ServiceStatus:         StatusGranted,
						StatusStartingTime:    "2026-01-01T00:00:00Z",
					},
				}},
			},
		},
	}
}

func TestParseLoTE_RoundTrip(t *testing.T) {
	lote := testLoTE()

	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	// Verify root wrapper
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasLoTE := raw["LoTE"]
	assert.True(t, hasLoTE, "JSON root must have LoTE key")

	parsed, err := ParseLoTE(data)
	require.NoError(t, err)

	assert.Equal(t, lote.ListAndSchemeInformation.LoTEVersionIdentifier, parsed.ListAndSchemeInformation.LoTEVersionIdentifier)
	assert.Equal(t, lote.ListAndSchemeInformation.SchemeTerritory, parsed.ListAndSchemeInformation.SchemeTerritory)
	assert.Equal(t, lote.ListAndSchemeInformation.LoTEType, parsed.ListAndSchemeInformation.LoTEType)
	assert.Equal(t, lote.ListAndSchemeInformation.ListIssueDateTime, parsed.ListAndSchemeInformation.ListIssueDateTime)
	assert.Equal(t, lote.ListAndSchemeInformation.NextUpdate, parsed.ListAndSchemeInformation.NextUpdate)
	assert.Equal(t, lote.ListAndSchemeInformation.LoTESequenceNumber, parsed.ListAndSchemeInformation.LoTESequenceNumber)

	require.Len(t, parsed.TrustedEntitiesList, 1)
	entity := parsed.TrustedEntitiesList[0]
	assert.Equal(t, "Example Issuer", entity.TrustedEntityInformation.TEName.Get("en", ""))
	require.Len(t, entity.TrustedEntityServices, 1)
	assert.Equal(t, StatusGranted, entity.TrustedEntityServices[0].ServiceInformation.ServiceStatus)
}

func TestParseLoTE_FromFile(t *testing.T) {
	lote := testLoTE()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	data, err := lote.MarshalIndent()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	parsed, err := ParseLoTEFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "SE", parsed.ListAndSchemeInformation.SchemeTerritory)
}

func TestParseLoTE_MissingRootKey(t *testing.T) {
	// JSON without the required "LoTE" root key
	_, err := ParseLoTE([]byte(`{"ListAndSchemeInformation": {}}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required")

	// Empty JSON object
	_, err = ParseLoTE([]byte(`{}`))
	assert.Error(t, err)

	// Invalid JSON
	_, err = ParseLoTE([]byte(`not json`))
	assert.Error(t, err)
}

func TestNameSet_Get(t *testing.T) {
	ns := NameSet{
		{Lang: "en", Value: "English"},
		{Lang: "sv", Value: "Svenska"},
	}

	assert.Equal(t, "English", ns.Get("en", ""))
	assert.Equal(t, "Svenska", ns.Get("sv", ""))
	assert.Equal(t, "English", ns.Get("de", ""))    // fallback to first
	assert.Equal(t, "", NameSet(nil).Get("en", "")) // nil set
}

func TestLoTE_JSONFieldNames(t *testing.T) {
	lote := testLoTE()
	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	s := string(data)
	// Verify official schema field names are present
	assert.Contains(t, s, `"LoTE"`)
	assert.Contains(t, s, `"ListAndSchemeInformation"`)
	assert.Contains(t, s, `"LoTEVersionIdentifier"`)
	assert.Contains(t, s, `"LoTESequenceNumber"`)
	assert.Contains(t, s, `"SchemeOperatorName"`)
	assert.Contains(t, s, `"SchemeTerritory"`)
	assert.Contains(t, s, `"ListIssueDateTime"`)
	assert.Contains(t, s, `"NextUpdate"`)
	assert.Contains(t, s, `"TrustedEntitiesList"`)
	assert.Contains(t, s, `"TrustedEntityInformation"`)
	assert.Contains(t, s, `"TrustedEntityServices"`)
	assert.Contains(t, s, `"ServiceInformation"`)
	assert.Contains(t, s, `"ServiceDigitalIdentity"`)
	assert.Contains(t, s, `"TEName"`)
	assert.Contains(t, s, `"TEInformationURI"`)
	assert.Contains(t, s, `"ServiceName"`)
	assert.Contains(t, s, `"ServiceStatus"`)
	assert.Contains(t, s, `"PublicKeyValues"`)

	// Verify multi-lang uses "lang" not "language"
	assert.Contains(t, s, `"lang"`)
	assert.NotContains(t, s, `"language"`)

	// Verify URI uses "uriValue" not "uri"
	assert.Contains(t, s, `"uriValue"`)

	// Verify OLD field names are NOT present
	assert.NotContains(t, s, `"schemeInformation"`)
	assert.NotContains(t, s, `"trustedEntities"`)
	assert.NotContains(t, s, `"version"`)
	assert.NotContains(t, s, `"entityId"`)
	assert.NotContains(t, s, `"entityStatus"`)
}

func TestLoTE_WithPointers(t *testing.T) {
	lote := &ListOfTrustedEntities{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    1,
			LoTEType:              LoTLTypeEU,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "EU"}},
			SchemeTerritory:       "EU",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
			PointersToOtherLoTE: []OtherLoTEPointer{
				{
					LoTELocation: "https://trust.example.se/pid.json",
					LoTEQualifiers: []LoTEQualifier{{
						LoTEType:           LoTETypePIDProviders,
						SchemeOperatorName: NameSet{{Lang: "en", Value: "Swedish Authority"}},
						SchemeTerritory:    "SE",
						MimeType:           "application/json",
					}},
				},
			},
		},
	}

	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	parsed, err := ParseLoTE(data)
	require.NoError(t, err)

	ptrs := parsed.ListAndSchemeInformation.PointersToOtherLoTE
	require.Len(t, ptrs, 1)
	assert.Equal(t, "https://trust.example.se/pid.json", ptrs[0].LoTELocation)
	require.Len(t, ptrs[0].LoTEQualifiers, 1)
	assert.Equal(t, LoTETypePIDProviders, ptrs[0].LoTEQualifiers[0].LoTEType)
	assert.Equal(t, "SE", ptrs[0].LoTEQualifiers[0].SchemeTerritory)
	assert.Equal(t, "application/json", ptrs[0].LoTEQualifiers[0].MimeType)

	s := string(data)
	assert.Contains(t, s, `"PointersToOtherLoTE"`)
	assert.Contains(t, s, `"LoTELocation"`)
	assert.Contains(t, s, `"LoTEQualifiers"`)
}

func TestLoTE_ServiceDigitalIdentity_X509(t *testing.T) {
	lote := &ListOfTrustedEntities{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "Test"}},
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
		},
		TrustedEntitiesList: []TrustedEntity{{
			TrustedEntityInformation: TrustedEntityInformation{
				TEName: NameSet{{Lang: "en", Value: "Test Entity"}},
			},
			TrustedEntityServices: []TrustedEntityService{{
				ServiceInformation: ServiceInformation{
					ServiceName: NameSet{{Lang: "en", Value: "Test"}},
					ServiceDigitalIdentity: ServiceDigitalIdentity{
						X509Certificates: []PKIOb{{Val: "MIIB...base64..."}},
						X509SubjectNames: []string{"CN=Test,O=Example"},
					},
					ServiceStatus: StatusGranted,
				},
			}},
		}},
	}

	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	parsed, err := ParseLoTE(data)
	require.NoError(t, err)

	sdi := parsed.TrustedEntitiesList[0].TrustedEntityServices[0].ServiceInformation.ServiceDigitalIdentity
	require.Len(t, sdi.X509Certificates, 1)
	assert.Equal(t, "MIIB...base64...", sdi.X509Certificates[0].Val)
	require.Len(t, sdi.X509SubjectNames, 1)
	assert.Equal(t, "CN=Test,O=Example", sdi.X509SubjectNames[0])
}

func TestEncodeToFile(t *testing.T) {
	lote := testLoTE()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	require.NoError(t, lote.EncodeToFile(path))

	parsed, err := ParseLoTEFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "SE", parsed.ListAndSchemeInformation.SchemeTerritory)
}

func TestMarshal(t *testing.T) {
	lote := testLoTE()
	data, err := lote.Marshal()
	require.NoError(t, err)

	parsed, err := ParseLoTE(data)
	require.NoError(t, err)
	assert.Equal(t, "SE", parsed.ListAndSchemeInformation.SchemeTerritory)
}

func TestParseMarshalLoTL(t *testing.T) {
	lotl := &ListOfTrustedLists{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    1,
			LoTEType:              LoTLTypeEU,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "EU"}},
			SchemeTerritory:       "EU",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
		},
	}

	data, err := lotl.MarshalLoTL()
	require.NoError(t, err)

	parsed, err := ParseLoTL(data)
	require.NoError(t, err)
	assert.Equal(t, "EU", parsed.ListAndSchemeInformation.SchemeTerritory)
	assert.True(t, parsed.IsLoTL())
}

func TestParseLoTLFromFile(t *testing.T) {
	lotl := &ListOfTrustedLists{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    1,
			LoTEType:              LoTLTypeEU,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "EU"}},
			SchemeTerritory:       "EU",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "lotl.json")
	data, err := lotl.MarshalLoTLIndent()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	parsed, err := ParseLoTLFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "EU", parsed.ListAndSchemeInformation.SchemeTerritory)
	assert.True(t, parsed.IsLoTL())
}
