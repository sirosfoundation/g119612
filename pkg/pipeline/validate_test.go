package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validLoTE(territory string) *etsi119602.ListOfTrustedEntities {
	return &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Test"}},
			LoTEType:              "http://example.com/type",
			SchemeTerritory:       territory,
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
		},
	}
}

func validLoTEWithEntity() *etsi119602.ListOfTrustedEntities {
	lote := validLoTE("SE")
	lote.TrustedEntitiesList = []etsi119602.TrustedEntity{
		{
			TrustedEntityInformation: etsi119602.TrustedEntityInformation{
				TEName: etsi119602.NameSet{{Lang: "en", Value: "Entity"}},
				TEAddress: &etsi119602.TEAddress{
					TEPostalAddress:     []etsi119602.PostalAddress{{Lang: "en", StreetAddress: "N/A", Country: "SE"}},
					TEElectronicAddress: []etsi119602.NonEmptyMultiLangURI{{Lang: "en", URIValue: "https://entity.example.com"}},
				},
				TEInformationURI: []etsi119602.NonEmptyMultiLangURI{{Lang: "en", URIValue: "https://entity.example.com"}},
			},
			TrustedEntityServices: []etsi119602.TrustedEntityService{{
				ServiceInformation: etsi119602.ServiceInformation{
					ServiceName: etsi119602.NameSet{{Lang: "en", Value: "Svc"}},
				},
			}},
		},
	}
	return lote
}

func testValidLoTL() *etsi119602.ListOfTrustedLists {
	return &etsi119602.ListOfTrustedLists{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "EC"}},
			LoTEType:              etsi119602.LoTLTypeEU,
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
			PointersToOtherLoTE: []etsi119602.OtherLoTEPointer{
				{LoTELocation: "https://example.com/lote.json"},
			},
		},
	}
}

func TestValidateLoTE_Valid(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(validLoTEWithEntity())

	ctx, err := ValidateLoTE(nil, ctx)
	require.NoError(t, err)
}

func TestValidateLoTE_Invalid(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{})

	_, err := ValidateLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidateLoTE_Empty(t *testing.T) {
	ctx := NewContext()
	_, err := ValidateLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LoTEs or LoTLs")
}

func TestValidateLoTE_MultipleOneInvalid(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(validLoTE("SE"))
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{})

	_, err := ValidateLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LoTE 1")
}

func TestValidateLoTL_Valid(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureLoTLs()
	ctx.AddLoTL(testValidLoTL())

	ctx, err := ValidateLoTL(nil, ctx)
	require.NoError(t, err)
}

func TestValidateLoTL_Invalid(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureLoTLs()
	ctx.AddLoTL(&etsi119602.ListOfTrustedLists{})

	_, err := ValidateLoTL(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidateLoTL_Empty(t *testing.T) {
	ctx := NewContext()
	_, err := ValidateLoTL(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LoTEs or LoTLs")
}

func TestValidate_Mixed(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(validLoTEWithEntity())
	ctx.EnsureLoTLs()
	ctx.AddLoTL(testValidLoTL())

	ctx, err := Validate(nil, ctx)
	require.NoError(t, err)
}
