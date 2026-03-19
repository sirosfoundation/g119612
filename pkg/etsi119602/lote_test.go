package etsi119602

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLoTE(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			Territory: "SE",
			SchemeOperator: NameSet{
				{Language: "en", Value: "Swedish Authority"},
			},
			SchemeType:     "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUlistoftrustedlists",
			IssueDate:      now,
			SequenceNumber: 42,
		},
		TrustedEntities: []TrustedEntity{
			{
				EntityID:     "https://issuer.example.com",
				EntityName:   NameSet{{Language: "en", Value: "Example Issuer"}},
				EntityType:   "credential-issuer",
				EntityStatus: StatusGranted,
				DigitalIdentities: []DigitalIdentity{
					{Type: "jwk", JWK: map[string]any{"kty": "EC", "crv": "P-256"}},
				},
			},
		},
	}

	// Marshal
	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	// Parse back
	parsed, err := ParseLoTE(data)
	require.NoError(t, err)

	assert.Equal(t, LoTEVersion, parsed.Version)
	assert.Equal(t, "SE", parsed.SchemeInformation.Territory)
	assert.Equal(t, 42, parsed.SchemeInformation.SequenceNumber)
	assert.Len(t, parsed.TrustedEntities, 1)

	entity := parsed.TrustedEntities[0]
	assert.Equal(t, "https://issuer.example.com", entity.EntityID)
	assert.Equal(t, StatusGranted, entity.EntityStatus)
	assert.Len(t, entity.DigitalIdentities, 1)
	assert.Equal(t, "jwk", entity.DigitalIdentities[0].Type)
}

func TestNameSet_Get(t *testing.T) {
	ns := NameSet{
		{Language: "en", Value: "English Name"},
		{Language: "sv", Value: "Svenskt Namn"},
	}

	assert.Equal(t, "English Name", ns.Get("en", "fallback"))
	assert.Equal(t, "Svenskt Namn", ns.Get("sv", "fallback"))
	assert.Equal(t, "English Name", ns.Get("de", "fallback")) // first item as fallback
}

func TestNameSet_Get_Empty(t *testing.T) {
	var ns NameSet
	assert.Equal(t, "fallback", ns.Get("en", "fallback"))
}

func TestLoTEPointers(t *testing.T) {
	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			SchemeType:     "http://example.com/lolote",
			SchemeOperator: NameSet{{Language: "en", Value: "EU"}},
			IssueDate:      time.Now().UTC(),
		},
		PointersToOtherLoTEs: []LoTEPointer{
			{Location: "https://se.example.com/lote.json", SchemeTerritory: "SE"},
			{Location: "https://de.example.com/lote.json", SchemeTerritory: "DE"},
		},
	}

	data, err := lote.Marshal()
	require.NoError(t, err)

	parsed, err := ParseLoTE(data)
	require.NoError(t, err)
	assert.Len(t, parsed.PointersToOtherLoTEs, 2)
	assert.Equal(t, "SE", parsed.PointersToOtherLoTEs[0].SchemeTerritory)
}

func TestParseLoTE_Invalid(t *testing.T) {
	_, err := ParseLoTE([]byte("not json"))
	assert.Error(t, err)
}

func TestParseLoTEFromFile_NotFound(t *testing.T) {
	_, err := ParseLoTEFromFile("/nonexistent/path.json")
	assert.Error(t, err)
}

func TestLoTE_JSONStructure(t *testing.T) {
	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			SchemeType:     "http://example.com/type",
			SchemeOperator: NameSet{{Language: "en", Value: "Ops"}},
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		TrustedEntities: []TrustedEntity{
			{
				EntityID:     "https://example.com",
				EntityName:   NameSet{{Language: "en", Value: "Test"}},
				EntityStatus: StatusGranted,
				Services: []EntityService{
					{
						ServiceType:   "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
						ServiceStatus: StatusGranted,
					},
				},
			},
		},
	}

	data, err := lote.Marshal()
	require.NoError(t, err)

	// Verify structure via generic map
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Equal(t, "1.0", m["version"])
	assert.NotNil(t, m["schemeInformation"])
	assert.NotNil(t, m["trustedEntities"])

	entities := m["trustedEntities"].([]any)
	assert.Len(t, entities, 1)

	entity := entities[0].(map[string]any)
	assert.Equal(t, "https://example.com", entity["entityId"])
	assert.NotNil(t, entity["services"])
}
