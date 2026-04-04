package jws

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewCertFileVerifier coverage ---

func TestNewCertFileVerifier_Valid(t *testing.T) {
	dir := t.TempDir()
	certPath, _ := generateTestCertAndKey(t, dir, "ec256")

	verifier, err := NewCertFileVerifier(certPath)
	require.NoError(t, err)
	assert.NotNil(t, verifier)
	assert.Len(t, verifier.keys, 1)
}

func TestNewCertFileVerifier_FileNotFound(t *testing.T) {
	_, err := NewCertFileVerifier("/nonexistent/cert.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read cert file")
}

func TestNewCertFileVerifier_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pem")
	require.NoError(t, os.WriteFile(bad, []byte("not a pem"), 0600))

	_, err := NewCertFileVerifier(bad)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no certificates found")
}

func TestNewCertFileVerifier_SignAndVerify(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "rsa")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	compact, err := signer.Sign([]byte(`{"test": true}`))
	require.NoError(t, err)

	verifier, err := NewCertFileVerifier(certPath)
	require.NoError(t, err)

	payload, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.JSONEq(t, `{"test": true}`, string(payload))
}

// --- algorithmForKey coverage (P-384, P-521, unsupported) ---

func TestAlgorithmForKey_EC384(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	alg, err := algorithmForKey(key)
	assert.NoError(t, err)
	assert.Equal(t, "ES384", string(alg))
}

func TestAlgorithmForKey_EC521(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)
	alg, err := algorithmForKey(key)
	assert.NoError(t, err)
	assert.Equal(t, "ES512", string(alg))
}

func TestAlgorithmForKey_Unsupported(t *testing.T) {
	_, err := algorithmForKey("not-a-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key type")
}

// --- parseCertificates coverage (non-CERTIFICATE blocks, empty PEM) ---

func TestParseCertificates_SkipsNonCertBlocks(t *testing.T) {
	// Create a PEM with both a private key block and a certificate block
	dir := t.TempDir()
	certPath, _ := generateTestCertAndKey(t, dir, "rsa")
	certPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)

	// Prepend a non-CERTIFICATE PEM block
	combined := append(
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte{0x01}}),
		certPEM...,
	)

	certs, err := parseCertificates(combined)
	require.NoError(t, err)
	assert.Len(t, certs, 1)
}

func TestParseCertificates_NoCerts(t *testing.T) {
	// PEM data with only non-CERTIFICATE blocks
	onlyKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte{0x01}})
	_, err := parseCertificates(onlyKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no certificates found")
}

func TestParseCertificates_EmptyInput(t *testing.T) {
	_, err := parseCertificates(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no certificates found")
}

func TestParseCertificates_MultipleCerts(t *testing.T) {
	dir := t.TempDir()
	// Generate two separate certs
	cert1 := generateSelfSignedCert(t, "cert1")
	cert2 := generateSelfSignedCert(t, "cert2")

	combined := filepath.Join(dir, "chain.pem")
	f, err := os.Create(combined)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert1.Raw}))
	require.NoError(t, pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert2.Raw}))
	f.Close()

	data, err := os.ReadFile(combined)
	require.NoError(t, err)
	certs, err := parseCertificates(data)
	require.NoError(t, err)
	assert.Len(t, certs, 2)
}

// --- FileSigner EC384 full roundtrip (already tested but covers algorithmForKey branch) ---

func TestFileSigner_EC384_Full(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "ec384")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	compact, err := signer.Sign([]byte("payload"))
	require.NoError(t, err)

	verifier, err := NewCertFileVerifier(certPath)
	require.NoError(t, err)

	payload, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), payload)
}

// --- helper ---

func generateSelfSignedCert(t *testing.T, cn string) *x509.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}
