//go:build softhsm

// Package selfsign exercises dsig.SelfSignCertificate against a real SoftHSM
// token.
//
// It lives in its own directory, and so its own test binary, on purpose. The
// PKCS#11 module can only be initialized once per process, and pkg/dsig's
// other tests initialize it against a token directory they then delete, which
// leaves any later crypto11.Configure in that binary failing with
// CKR_GENERAL_ERROR. pkg/jws and pkg/pipeline get away with their SoftHSM
// tests for the same reason: separate package, separate process.
package selfsign

import (
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"

	"github.com/ThalesGroup/crypto11"
	"github.com/sirosfoundation/g119612/pkg/dsig"
	"github.com/sirosfoundation/g119612/pkg/dsig/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	selfSignKeyLabel  = "test-key"
	selfSignCertLabel = "test-cert"
	selfSignKeyID     = "01"
)

var selfSignKeyIDBytes = []byte{0x01}

// TestSelfSignCertificate covers issuing a certificate for a key that already
// lives in a PKCS#11 token.
//
// The subtests share one token and one crypto11 context deliberately: the
// PKCS#11 module cannot be initialized more than once per process, so a helper
// that configured a fresh context per test would fail with CKR_GENERAL_ERROR.
// They therefore run in order, and the first one depends on the token still
// being in its initial state.
func TestSelfSignCertificate(t *testing.T) {
	helper := test.SkipIfSoftHSMUnavailable(t)
	require.NoError(t, helper.Setup())
	t.Cleanup(func() { _ = helper.Cleanup() })

	if err := helper.GenerateAndImportTestCert(selfSignKeyLabel, selfSignCertLabel, selfSignKeyID); err != nil {
		t.Skipf("could not set up SoftHSM token: %v", err)
	}

	config := dsig.ExtractPKCS11Config(helper.GetPKCS11URI())
	require.NotNil(t, config)

	ctx, err := crypto11.Configure(config)
	require.NoError(t, err)

	keyPair, err := ctx.FindKeyPair(selfSignKeyIDBytes, []byte(selfSignKeyLabel))
	require.NoError(t, err)
	require.NotNil(t, keyPair)

	type equaler interface{ Equal(x crypto.PublicKey) bool }
	tokenKey, ok := keyPair.Public().(equaler)
	require.True(t, ok, "token public key must support Equal")

	tokenCert := func(t *testing.T) *x509.Certificate {
		t.Helper()
		c, err := ctx.FindCertificate(selfSignKeyIDBytes, []byte(selfSignCertLabel), nil)
		require.NoError(t, err)
		require.NotNil(t, c)
		return c
	}

	// This is the regression this whole file exists for. GenerateAndImportTestCert
	// sets the token up the way the documented shell recipe does: generate a key
	// pair in the token, then mint a certificate over a separate, throwaway key
	// and load that. Certificate and signing key then describe different keys,
	// and every signature produced is unverifiable by anyone holding the
	// certificate — with no error at signing time.
	t.Run("fixture starts out mismatched", func(t *testing.T) {
		assert.False(t, tokenKey.Equal(tokenCert(t).PublicKey),
			"expected the fixture's certificate to disagree with the token key")
	})

	t.Run("without cert label the token is left alone", func(t *testing.T) {
		before := tokenCert(t).Raw

		cert, _, err := dsig.SelfSignCertificate(config, dsig.SelfSignedCertOptions{
			KeyLabel: selfSignKeyLabel,
			KeyID:    selfSignKeyID,
			Subject:  pkix.Name{CommonName: "Self Sign Test"},
			Validity: 24 * time.Hour,
		})
		require.NoError(t, err)

		assert.Equal(t, before, tokenCert(t).Raw, "token certificate must be untouched")
		assert.NotEqual(t, cert.Raw, before)
		// Even when not stored, the certificate must describe the token key.
		assert.True(t, tokenKey.Equal(cert.PublicKey))
	})

	t.Run("reissued certificate matches the token key", func(t *testing.T) {
		cert, pemBytes, err := dsig.SelfSignCertificate(config, dsig.SelfSignedCertOptions{
			KeyLabel:  selfSignKeyLabel,
			KeyID:     selfSignKeyID,
			CertLabel: selfSignCertLabel,
			Subject:   pkix.Name{CommonName: "Self Sign Test"},
			Validity:  24 * time.Hour,
		})
		require.NoError(t, err)
		assert.Contains(t, string(pemBytes), "BEGIN CERTIFICATE")

		assert.True(t, tokenKey.Equal(cert.PublicKey),
			"certificate public key must match the token's signing key")
		assert.NoError(t, cert.CheckSignatureFrom(cert), "certificate must be self-consistent")
		assert.Equal(t, "Self Sign Test", cert.Subject.CommonName)
		assert.True(t, cert.IsCA)
		assert.Equal(t,
			x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign|x509.KeyUsageCRLSign,
			cert.KeyUsage)

		// The token now hands out that certificate, not the stale one, and there
		// is exactly one certificate under the label.
		stored := tokenCert(t)
		assert.Equal(t, cert.Raw, stored.Raw, "token must return the reissued certificate")
		assert.True(t, tokenKey.Equal(stored.PublicKey))
	})

	t.Run("missing key is an error", func(t *testing.T) {
		_, _, err := dsig.SelfSignCertificate(config, dsig.SelfSignedCertOptions{
			KeyLabel: "no-such-key",
			KeyID:    selfSignKeyID,
			Subject:  pkix.Name{CommonName: "Self Sign Test"},
			Validity: 24 * time.Hour,
		})
		require.Error(t, err)
	})
}

// Input validation happens before the token is touched, so this needs no HSM.
func TestSelfSignCertificateValidatesInput(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		_, _, err := dsig.SelfSignCertificate(nil, dsig.SelfSignedCertOptions{
			KeyLabel: "k", KeyID: "01", Validity: time.Hour,
		})
		require.Error(t, err)
	})

	cases := map[string]dsig.SelfSignedCertOptions{
		"no key label":      {KeyID: "01", Validity: time.Hour},
		"zero validity":     {KeyLabel: "k", KeyID: "01"},
		"negative validity": {KeyLabel: "k", KeyID: "01", Validity: -time.Hour},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := dsig.SelfSignCertificate(&crypto11.Config{Path: "/nonexistent"}, opts)
			require.Error(t, err)
		})
	}
}
