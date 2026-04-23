package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeLoTEs_Basic(t *testing.T) {
	ctx := NewContext()
	si := validSchemeInfo("SE")
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: si,
		TrustedEntitiesList: []etsi119602.TrustedEntity{
			{TrustedEntityInformation: etsi119602.TrustedEntityInformation{TEName: etsi119602.NameSet{{Lang: "en", Value: "entity-1"}}}, TrustedEntityServices: []etsi119602.TrustedEntityService{{ServiceInformation: etsi119602.ServiceInformation{ServiceName: etsi119602.NameSet{{Lang: "en", Value: "svc"}}, ServiceStatus: etsi119602.StatusGranted}}}},
		},
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: validSchemeInfo("NO"),
		TrustedEntitiesList: []etsi119602.TrustedEntity{
			{TrustedEntityInformation: etsi119602.TrustedEntityInformation{TEName: etsi119602.NameSet{{Lang: "en", Value: "entity-2"}}}, TrustedEntityServices: []etsi119602.TrustedEntityService{{ServiceInformation: etsi119602.ServiceInformation{ServiceName: etsi119602.NameSet{{Lang: "en", Value: "svc"}}, ServiceStatus: etsi119602.StatusGranted}}}},
			{TrustedEntityInformation: etsi119602.TrustedEntityInformation{TEName: etsi119602.NameSet{{Lang: "en", Value: "entity-3"}}}, TrustedEntityServices: []etsi119602.TrustedEntityService{{ServiceInformation: etsi119602.ServiceInformation{ServiceName: etsi119602.NameSet{{Lang: "en", Value: "svc"}}, ServiceStatus: etsi119602.StatusGranted}}}},
		},
	})

	ctx, err := MergeLoTEs(nil, ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	merged := ctx.GetLoTEs()[0]
	assert.Len(t, merged.TrustedEntitiesList, 3)
	assert.Equal(t, "SE", merged.ListAndSchemeInformation.SchemeTerritory) // Takes first LoTE's scheme info
}

func TestMergeLoTEs_SingleDoesNothing(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: validSchemeInfo("SE"),
	})

	ctx, err := MergeLoTEs(nil, ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, ctx.GetLoTECount())
}

func TestMergeLoTEs_Empty(t *testing.T) {
	ctx := NewContext()
	_, err := MergeLoTEs(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LoTEs")
}

func TestMergeLoTEs_Pointers(t *testing.T) {
	ctx := NewContext()
	si1 := validSchemeInfo("SE")
	si1.PointersToOtherLoTE = []etsi119602.OtherLoTEPointer{
		{LoTELocation: "https://se.example.com/lote.json"},
	}
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: si1,
	})
	si2 := validSchemeInfo("NO")
	si2.PointersToOtherLoTE = []etsi119602.OtherLoTEPointer{
		{LoTELocation: "https://no.example.com/lote.json"},
	}
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: si2,
	})

	ctx, err := MergeLoTEs(nil, ctx)
	require.NoError(t, err)
	assert.Len(t, ctx.GetLoTEs()[0].ListAndSchemeInformation.PointersToOtherLoTE, 2)
}

func TestIncrementLoTESequence(t *testing.T) {
	ctx := NewContext()
	si := validSchemeInfo("SE")
	si.LoTESequenceNumber = 5
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: si,
	})

	ctx, err := IncrementLoTESequence(nil, ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, ctx.GetLoTEs()[0].ListAndSchemeInformation.LoTESequenceNumber)
}

func TestIncrementLoTESequence_Empty(t *testing.T) {
	ctx := NewContext()
	_, err := IncrementLoTESequence(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LoTEs")
}

func TestIncrementLoTESequence_Multiple(t *testing.T) {
	ctx := NewContext()
	si1 := validSchemeInfo("SE")
	si1.LoTESequenceNumber = 10
	si2 := validSchemeInfo("NO")
	si2.LoTESequenceNumber = 20
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: si1,
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: si2,
	})

	ctx, err := IncrementLoTESequence(nil, ctx)
	require.NoError(t, err)
	lotes := ctx.GetLoTEs()
	assert.Equal(t, 11, lotes[0].ListAndSchemeInformation.LoTESequenceNumber)
	assert.Equal(t, 21, lotes[1].ListAndSchemeInformation.LoTESequenceNumber)
}
