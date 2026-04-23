package etsi119602

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXML_RoundTrip(t *testing.T) {
	lote := testLoTE()

	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), "ListOfTrustedEntities")

	parsed, err := ParseLoTEXML(xmlData)
	require.NoError(t, err)

	assert.Equal(t, lote.ListAndSchemeInformation.SchemeTerritory, parsed.ListAndSchemeInformation.SchemeTerritory)
	assert.Equal(t, lote.ListAndSchemeInformation.LoTEType, parsed.ListAndSchemeInformation.LoTEType)
	require.Len(t, parsed.TrustedEntitiesList, 1)

	entity := parsed.TrustedEntitiesList[0]
	assert.Equal(t, "Example Issuer", entity.TrustedEntityInformation.TEName.Get("en", ""))
	require.Len(t, entity.TrustedEntityServices, 1)
	assert.Equal(t, StatusGranted, entity.TrustedEntityServices[0].ServiceInformation.ServiceStatus)
}

func TestXML_FileRoundTrip(t *testing.T) {
	lote := testLoTE()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xml")

	require.NoError(t, lote.EncodeXMLToFile(path))

	parsed, err := ParseLoTEXMLFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "SE", parsed.ListAndSchemeInformation.SchemeTerritory)
}

func TestXML_WithPointers(t *testing.T) {
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
					ServiceDigitalIdentities: []ServiceDigitalIdentity{{
						X509Certificates: []PKIOb{{Val: "MIIB..."}},
					}},
				},
			},
		},
	}

	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)

	parsed, err := ParseLoTEXML(xmlData)
	require.NoError(t, err)

	ptrs := parsed.ListAndSchemeInformation.PointersToOtherLoTE
	require.Len(t, ptrs, 1)
	assert.Equal(t, "https://trust.example.se/pid.json", ptrs[0].LoTELocation)
}

func TestXML_LOTETag(t *testing.T) {
	lote := testLoTE()
	lote.LOTETag = "http://custom/tag"

	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), "http://custom/tag")

	parsed, err := ParseLoTEXML(xmlData)
	require.NoError(t, err)
	assert.Equal(t, "http://custom/tag", parsed.LOTETag)
}

func TestXML_ParseIDUnion(t *testing.T) {
	// The IDUnion XML has several non-conformances with the ETSI TS 119 602-1 XSD:
	// - Wrong root element: <TrustedEntitiesList> instead of <ListOfTrustedEntities>
	// - Root element not in the LoTE namespace
	// - DigitalId elements contain X509Certificate + empty X509SubjectName + empty X509SKI
	//   simultaneously, violating xsd:choice (only one should be present)
	// - Empty LOTETag attribute (required per XSD)
	// - Name elements wrap text in <NonEmptyNormalizedString> child elements
	//   (should be simple content per XSD)
	// - Empty <StreetAddress/> and <URI/> elements (violate minLength=1)
	// - Invalid URIs: "https://htpps://www.very-good-wallet.com"
	// - Two ds:Signature elements (XSD allows at most one)
	// - First signature has empty <Object/> elements (broken XAdES)
	// Despite these issues, our parser should still be able to extract the data.
	data, err := os.ReadFile("testdata/idunion_lote.xml")
	require.NoError(t, err)

	lote, err := ParseLoTEXML(data)
	require.NoError(t, err)

	assert.NotEmpty(t, lote.ListAndSchemeInformation.SchemeOperatorName)
	assert.NotEmpty(t, lote.TrustedEntitiesList)
	assert.Equal(t, LoTETypeWalletProviders, lote.ListAndSchemeInformation.LoTEType)
	assert.GreaterOrEqual(t, len(lote.TrustedEntitiesList), 4) // At least 4 entities
}

func TestXML_LoTLRoundTrip(t *testing.T) {
	lotl := &ListOfTrustedLists{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    1,
			LoTEType:              LoTLTypeEU,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "EU"}},
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
			PointersToOtherLoTE: []OtherLoTEPointer{
				{LoTELocation: "https://example.com/lote.json"},
			},
		},
	}

	xmlData, err := lotl.EncodeXML()
	require.NoError(t, err)

	parsed, err := ParseLoTLXML(xmlData)
	require.NoError(t, err)

	ptrs := parsed.ListAndSchemeInformation.PointersToOtherLoTE
	require.Len(t, ptrs, 1)
	assert.Equal(t, "https://example.com/lote.json", ptrs[0].LoTELocation)
}

func TestXML_PolicyRoundTrip(t *testing.T) {
	lote := testLoTE()
	lote.ListAndSchemeInformation.PolicyOrLegalNotice = []PolicyOrLegalNoticeItem{
		{LoTEPolicy: &NonEmptyMultiLangURI{Lang: "en", URIValue: "https://policy.example.com"}},
		{LoTELegalNotice: "Legal notice text"},
	}

	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)

	parsed, err := ParseLoTEXML(xmlData)
	require.NoError(t, err)

	require.Len(t, parsed.ListAndSchemeInformation.PolicyOrLegalNotice, 2)
	assert.NotNil(t, parsed.ListAndSchemeInformation.PolicyOrLegalNotice[0].LoTEPolicy)
	assert.Equal(t, "https://policy.example.com", parsed.ListAndSchemeInformation.PolicyOrLegalNotice[0].LoTEPolicy.URIValue)
	assert.Equal(t, "Legal notice text", parsed.ListAndSchemeInformation.PolicyOrLegalNotice[1].LoTELegalNotice)
}
