package etsi119602

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validLoTE() *ListOfTrustedEntities {
	return &ListOfTrustedEntities{
		Version: "1.0",
		SchemeInformation: SchemeInformation{
			SchemeOperator: NameSet{{Language: "en", Value: "Test Operator"}},
			SchemeType:     "http://example.com/type",
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestValidate_ValidMinimal(t *testing.T) {
	lote := validLoTE()
	assert.NoError(t, lote.Validate())
}

func TestValidate_MissingVersion(t *testing.T) {
	lote := validLoTE()
	lote.Version = ""
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestValidate_MissingSchemeOperator(t *testing.T) {
	lote := validLoTE()
	lote.SchemeInformation.SchemeOperator = nil
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schemeOperator is required")
}

func TestValidate_MissingSchemeType(t *testing.T) {
	lote := validLoTE()
	lote.SchemeInformation.SchemeType = ""
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schemeType is required")
}

func TestValidate_MissingIssueDate(t *testing.T) {
	lote := validLoTE()
	lote.SchemeInformation.IssueDate = time.Time{}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issueDate is required")
}

func TestValidate_EntityMissingID(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{EntityStatus: StatusGranted}}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entityId is required")
}

func TestValidate_EntityMissingStatus(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{EntityID: "urn:test"}}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entityStatus is required")
}

func TestValidate_DigitalIdentityJWK(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{
		EntityID:     "urn:test",
		EntityStatus: StatusGranted,
		DigitalIdentities: []DigitalIdentity{
			{Type: "jwk", JWK: map[string]any{"kty": "EC"}},
		},
	}}
	assert.NoError(t, lote.Validate())
}

func TestValidate_DigitalIdentityJWKEmpty(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{
		EntityID:     "urn:test",
		EntityStatus: StatusGranted,
		DigitalIdentities: []DigitalIdentity{
			{Type: "jwk"},
		},
	}}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwk must not be empty")
}

func TestValidate_DigitalIdentityDID(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{
		EntityID:     "urn:test",
		EntityStatus: StatusGranted,
		DigitalIdentities: []DigitalIdentity{
			{Type: "did", DID: "did:web:example.com"},
		},
	}}
	assert.NoError(t, lote.Validate())
}

func TestValidate_DigitalIdentityDIDInvalid(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{
		EntityID:     "urn:test",
		EntityStatus: StatusGranted,
		DigitalIdentities: []DigitalIdentity{
			{Type: "did", DID: "not-a-did"},
		},
	}}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did must start with")
}

func TestValidate_DigitalIdentityUnknownType(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{
		EntityID:     "urn:test",
		EntityStatus: StatusGranted,
		DigitalIdentities: []DigitalIdentity{
			{Type: "unknown"},
		},
	}}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

func TestValidate_ServiceMissingType(t *testing.T) {
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{
		EntityID:     "urn:test",
		EntityStatus: StatusGranted,
		Services:     []EntityService{{ServiceStatus: StatusGranted}},
	}}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serviceType is required")
}

func TestValidate_PointerMissingLocation(t *testing.T) {
	lote := validLoTE()
	lote.PointersToOtherLoTEs = []LoTEPointer{{}}
	err := lote.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "location is required")
}

func TestValidate_ValidFull(t *testing.T) {
	now := time.Now()
	lote := validLoTE()
	lote.TrustedEntities = []TrustedEntity{{
		EntityID:     "urn:test",
		EntityStatus: StatusGranted,
		DigitalIdentities: []DigitalIdentity{
			{Type: "x509", X509Certificate: "MIIB..."},
			{Type: "x509_subject_name", X509SubjectName: "CN=Test"},
		},
		Services: []EntityService{{
			ServiceType:   "http://example.com/svc",
			ServiceStatus: StatusGranted,
		}},
		StatusStartingTime: &now,
	}}
	lote.PointersToOtherLoTEs = []LoTEPointer{{
		Location: "https://example.com/other.json",
	}}
	assert.NoError(t, lote.Validate())
}
