package pipeline

import (
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLoTE_Valid(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version: "1.0",
		SchemeInformation: etsi119602.SchemeInformation{
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Test"}},
			SchemeType:     "http://example.com/type",
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		TrustedEntities: []etsi119602.TrustedEntity{
			{EntityID: "https://example.com", EntityStatus: etsi119602.StatusGranted},
		},
	})

	ctx, err := ValidateLoTE(nil, ctx)
	require.NoError(t, err)
}

func TestValidateLoTE_Invalid(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		// Missing version
		SchemeInformation: etsi119602.SchemeInformation{
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Test"}},
			SchemeType:     "http://example.com/type",
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	})

	_, err := ValidateLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidateLoTE_Empty(t *testing.T) {
	ctx := NewContext()
	_, err := ValidateLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LoTEs")
}

func TestValidateLoTE_MultipleOneInvalid(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version: "1.0",
		SchemeInformation: etsi119602.SchemeInformation{
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Valid"}},
			SchemeType:     "http://example.com/type",
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		// Invalid: missing version
	})

	_, err := ValidateLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LoTE 1")
}

func TestValidateLoTL_Valid(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureLoTLs()
	ctx.AddLoTL(&etsi119602.ListOfTrustedLists{
		Version: etsi119602.LoTEVersion,
		SchemeInformation: etsi119602.SchemeInformation{
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "EC"}},
			SchemeType:     etsi119602.LoTLTypeEU,
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		PointersToOtherLoTEs: []etsi119602.LoTEPointer{
			{Location: "https://example.com/lote.json"},
		},
	})

	ctx, err := ValidateLoTL(nil, ctx)
	require.NoError(t, err)
}

func TestValidateLoTL_Invalid(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureLoTLs()
	ctx.AddLoTL(&etsi119602.ListOfTrustedLists{
		// Missing version
	})

	_, err := ValidateLoTL(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidateLoTL_Empty(t *testing.T) {
	ctx := NewContext()
	_, err := ValidateLoTL(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LoTLs")
}
