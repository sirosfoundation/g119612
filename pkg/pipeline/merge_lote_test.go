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
		Version:           "1.0",
		SchemeInformation: si,
		TrustedEntities: []etsi119602.TrustedEntity{
			{EntityID: "entity-1", EntityStatus: etsi119602.StatusGranted},
		},
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("NO"),
		TrustedEntities: []etsi119602.TrustedEntity{
			{EntityID: "entity-2", EntityStatus: etsi119602.StatusGranted},
			{EntityID: "entity-3", EntityStatus: etsi119602.StatusGranted},
		},
	})

	ctx, err := MergeLoTEs(nil, ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	merged := ctx.GetLoTEs()[0]
	assert.Len(t, merged.TrustedEntities, 3)
	assert.Equal(t, "SE", merged.SchemeInformation.Territory) // Takes first LoTE's scheme info
}

func TestMergeLoTEs_SingleDoesNothing(t *testing.T) {
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("SE"),
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
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("SE"),
		PointersToOtherLoTEs: []etsi119602.LoTEPointer{
			{Location: "https://se.example.com/lote.json"},
		},
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("NO"),
		PointersToOtherLoTEs: []etsi119602.LoTEPointer{
			{Location: "https://no.example.com/lote.json"},
		},
	})

	ctx, err := MergeLoTEs(nil, ctx)
	require.NoError(t, err)
	assert.Len(t, ctx.GetLoTEs()[0].PointersToOtherLoTEs, 2)
}

func TestIncrementLoTESequence(t *testing.T) {
	ctx := NewContext()
	si := validSchemeInfo("SE")
	si.SequenceNumber = 5
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: si,
	})

	ctx, err := IncrementLoTESequence(nil, ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, ctx.GetLoTEs()[0].SchemeInformation.SequenceNumber)
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
	si1.SequenceNumber = 10
	si2 := validSchemeInfo("NO")
	si2.SequenceNumber = 20
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: si1,
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: si2,
	})

	ctx, err := IncrementLoTESequence(nil, ctx)
	require.NoError(t, err)
	lotes := ctx.GetLoTEs()
	assert.Equal(t, 11, lotes[0].SchemeInformation.SequenceNumber)
	assert.Equal(t, 21, lotes[1].SchemeInformation.SequenceNumber)
}


