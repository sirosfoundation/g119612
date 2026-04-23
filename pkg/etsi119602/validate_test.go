package etsi119602

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate_Valid(t *testing.T) {
	lote := testLoTE()
	assert.NoError(t, lote.Validate())
}

func TestValidate_MissingSchemeOperatorName(t *testing.T) {
	lote := testLoTE()
	lote.ListAndSchemeInformation.SchemeOperatorName = nil
	err := lote.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SchemeOperatorName")
}

func TestValidate_MissingListIssueDateTime(t *testing.T) {
	lote := testLoTE()
	lote.ListAndSchemeInformation.ListIssueDateTime = ""
	err := lote.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ListIssueDateTime")
}

func TestValidate_MissingNextUpdate(t *testing.T) {
	lote := testLoTE()
	lote.ListAndSchemeInformation.NextUpdate = ""
	err := lote.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NextUpdate")
}

func TestValidate_EntityMissingTEName(t *testing.T) {
	lote := testLoTE()
	lote.TrustedEntitiesList[0].TrustedEntityInformation.TEName = nil
	err := lote.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TEName")
}

func TestValidate_EntityMissingServices(t *testing.T) {
	lote := testLoTE()
	lote.TrustedEntitiesList[0].TrustedEntityServices = nil
	err := lote.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TrustedEntityServices")
}

func TestValidate_ServiceMissingServiceName(t *testing.T) {
	lote := testLoTE()
	lote.TrustedEntitiesList[0].TrustedEntityServices[0].ServiceInformation.ServiceName = nil
	err := lote.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ServiceName")
}

func TestValidate_PointerMissingLocation(t *testing.T) {
	lote := &ListOfTrustedEntities{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "Test"}},
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
			PointersToOtherLoTE: []OtherLoTEPointer{
				{LoTELocation: ""},
			},
		},
	}
	err := lote.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LoTELocation")
}
