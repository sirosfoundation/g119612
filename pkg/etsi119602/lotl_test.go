package etsi119602

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLoTLSchemeType(t *testing.T) {
	assert.True(t, IsLoTLSchemeType(LoTLTypeEU))
	assert.False(t, IsLoTLSchemeType(LoTETypePIDProviders))
	assert.False(t, IsLoTLSchemeType(""))
}

func TestLoTE_IsLoTL(t *testing.T) {
	lotl := &ListOfTrustedEntities{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEType: LoTLTypeEU,
		},
	}
	assert.True(t, lotl.IsLoTL())

	lote := &ListOfTrustedEntities{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEType: LoTETypePIDProviders,
		},
	}
	assert.False(t, lote.IsLoTL())
}

func TestParseLoTL_RoundTrip(t *testing.T) {
	lotl := &ListOfTrustedLists{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    1,
			LoTEType:              LoTLTypeEU,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "EU Commission"}},
			SchemeTerritory:       "EU",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
			PointersToOtherLoTE: []OtherLoTEPointer{
				{
					LoTELocation: "https://trust.example.se/pid.json",
					LoTEQualifiers: []LoTEQualifier{{
						LoTEType:           LoTETypePIDProviders,
						SchemeOperatorName: NameSet{{Lang: "en", Value: "Swedish PID Authority"}},
						SchemeTerritory:    "SE",
						MimeType:           "application/json",
					}},
				},
				{
					LoTELocation: "https://trust.example.de/pid.json",
					LoTEQualifiers: []LoTEQualifier{{
						LoTEType:           LoTETypePIDProviders,
						SchemeOperatorName: NameSet{{Lang: "en", Value: "German PID Authority"}},
						SchemeTerritory:    "DE",
						MimeType:           "application/json",
					}},
				},
			},
		},
	}

	data, err := lotl.MarshalIndent()
	require.NoError(t, err)

	parsed, err := ParseLoTL(data)
	require.NoError(t, err)
	assert.True(t, parsed.IsLoTL())
	assert.Equal(t, "EU", parsed.ListAndSchemeInformation.SchemeTerritory)

	ptrs := parsed.ListAndSchemeInformation.PointersToOtherLoTE
	require.Len(t, ptrs, 2)
	assert.Equal(t, "https://trust.example.se/pid.json", ptrs[0].LoTELocation)
	assert.Equal(t, "SE", ptrs[0].LoTEQualifiers[0].SchemeTerritory)
}

func TestValidate_LoTL(t *testing.T) {
	lotl := &ListOfTrustedLists{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "Test"}},
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
			PointersToOtherLoTE: []OtherLoTEPointer{
				{LoTELocation: "https://example.com/lote.json"},
			},
		},
	}
	assert.NoError(t, lotl.Validate())
}
