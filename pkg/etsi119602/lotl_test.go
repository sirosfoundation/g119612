package etsi119602

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoTLRoundTrip(t *testing.T) {
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
			{
				Location:        "https://example.com/lote-wallet.json",
				SchemeTerritory: "EU",
				SchemeType:      LoTETypeWalletProviders,
			},
		},
	}

	// Marshal to JSON
	data, err := lotl.MarshalIndent()
	require.NoError(t, err)

	// Parse back
	parsed, err := ParseLoTL(data)
	require.NoError(t, err)

	assert.Equal(t, LoTEVersion, parsed.Version)
	assert.Equal(t, "EU", parsed.SchemeInformation.Territory)
	assert.Equal(t, LoTLTypeEU, parsed.SchemeInformation.SchemeType)
	assert.Len(t, parsed.PointersToOtherLoTEs, 2)
	assert.Equal(t, "https://example.com/lote-pid.json", parsed.PointersToOtherLoTEs[0].Location)
	assert.Equal(t, LoTETypePIDProviders, parsed.PointersToOtherLoTEs[0].SchemeType)
}

func TestLoTLValidate(t *testing.T) {
	now := time.Now().UTC()

	t.Run("valid", func(t *testing.T) {
		lotl := &ListOfTrustedLists{
			Version: LoTEVersion,
			SchemeInformation: SchemeInformation{
				SchemeOperator: NameSet{{Language: "en", Value: "EU"}},
				SchemeType:     LoTLTypeEU,
				IssueDate:      now,
			},
			PointersToOtherLoTEs: []LoTEPointer{
				{Location: "https://example.com/lote.json"},
			},
		}
		assert.NoError(t, lotl.Validate())
	})

	t.Run("missing version", func(t *testing.T) {
		lotl := &ListOfTrustedLists{
			SchemeInformation: SchemeInformation{
				SchemeOperator: NameSet{{Language: "en", Value: "EU"}},
				SchemeType:     LoTLTypeEU,
				IssueDate:      now,
			},
		}
		assert.Error(t, lotl.Validate())
	})

	t.Run("invalid pointer", func(t *testing.T) {
		lotl := &ListOfTrustedLists{
			Version: LoTEVersion,
			SchemeInformation: SchemeInformation{
				SchemeOperator: NameSet{{Language: "en", Value: "EU"}},
				SchemeType:     LoTLTypeEU,
				IssueDate:      now,
			},
			PointersToOtherLoTEs: []LoTEPointer{
				{Location: ""}, // missing location
			},
		}
		assert.Error(t, lotl.Validate())
	})
}

func TestLoTLToLoTE(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	lotl := &ListOfTrustedLists{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			Territory: "EU",
			SchemeOperator: NameSet{
				{Language: "en", Value: "EC"},
			},
			SchemeType: LoTLTypeEU,
			IssueDate:  now,
		},
		PointersToOtherLoTEs: []LoTEPointer{
			{Location: "https://example.com/lote.json"},
		},
	}

	lote := lotl.ToLoTE()
	assert.Equal(t, LoTEVersion, lote.Version)
	assert.Equal(t, "EU", lote.SchemeInformation.Territory)
	assert.Nil(t, lote.TrustedEntities)
	assert.Len(t, lote.PointersToOtherLoTEs, 1)
}

func TestLoTLFromLoTE(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			Territory: "EU",
			SchemeOperator: NameSet{
				{Language: "en", Value: "EC"},
			},
			SchemeType: LoTLTypeEU,
			IssueDate:  now,
		},
		PointersToOtherLoTEs: []LoTEPointer{
			{Location: "https://example.com/lote.json"},
		},
	}

	lotl := LoTLFromLoTE(lote)
	assert.Equal(t, LoTEVersion, lotl.Version)
	assert.Equal(t, "EU", lotl.SchemeInformation.Territory)
	assert.Len(t, lotl.PointersToOtherLoTEs, 1)
}
