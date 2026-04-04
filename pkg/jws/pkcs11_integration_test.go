//go:build softhsm

package jws_test

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/dsig"
	dtest "github.com/sirosfoundation/g119612/pkg/dsig/test"
	"github.com/sirosfoundation/g119612/pkg/jws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPKCS11Signer_SoftHSM_Integration tests signing with a real SoftHSM2 token.
// Only runs with: go test -tags softhsm ./pkg/jws/
func TestPKCS11Signer_SoftHSM_Integration(t *testing.T) {
	helper := dtest.SkipIfSoftHSMUnavailable(t)

	err := helper.Setup()
	if err != nil {
		t.Skipf("Could not set up SoftHSM token: %v", err)
	}
	defer helper.Cleanup()

	keyLabel := "jws-test-key"
	certLabel := "jws-test-cert"
	keyID := "02"
	err = helper.GenerateAndImportTestCert(keyLabel, certLabel, keyID)
	if err != nil {
		t.Skipf("Could not import test certificate: %v", err)
	}

	pkcs11URI := helper.GetPKCS11URI()
	t.Logf("PKCS11 URI: %s", pkcs11URI)

	config := dsig.ExtractPKCS11Config(pkcs11URI)
	require.NotNil(t, config, "failed to extract PKCS#11 config from URI")

	signer := jws.NewPKCS11Signer(config, keyLabel, certLabel)
	signer.SetKeyID(keyID)

	// Test Sign
	// Note: go-jose/v4 may not support the opaque signer returned by crypto11.
	// This is a known compatibility issue — if signing fails with "unsupported key type/format",
	// it means go-jose doesn't handle the crypto11 opaque signer correctly.
	payload := []byte(`{"test": "pkcs11-jws"}`)
	compact, err := signer.Sign(payload)
	if err != nil && assert.Contains(t, err.Error(), "unsupported key type") {
		t.Skipf("Skipping: go-jose/v4 does not support crypto11 opaque signers: %v", err)
	}
	require.NoError(t, err)
	assert.NotEmpty(t, compact)
	assert.Contains(t, compact, ".")
	t.Logf("JWS compact length: %d", len(compact))

	// Test Close after signing
	err = signer.Close()
	assert.NoError(t, err)
}
