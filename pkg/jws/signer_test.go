package jws

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCertAndKey(t *testing.T, dir string, keyType string) (certPath, keyPath string) {
	t.Helper()

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	var privKey interface{}
	var pubKey interface{}

	switch keyType {
	case "rsa":
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		privKey = rsaKey
		pubKey = &rsaKey.PublicKey
	case "ec256":
		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		privKey = ecKey
		pubKey = &ecKey.PublicKey
	case "ec384":
		ecKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)
		privKey = ecKey
		pubKey = &ecKey.PublicKey
	default:
		t.Fatalf("unknown key type: %s", keyType)
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pubKey, privKey)
	require.NoError(t, err)

	certPath = filepath.Join(dir, "cert.pem")
	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	certFile.Close()

	keyPath = filepath.Join(dir, "key.pem")
	keyFile, err := os.Create(keyPath)
	require.NoError(t, err)

	keyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(keyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
	keyFile.Close()

	return certPath, keyPath
}

func TestFileSigner_RSA_SignVerify(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "rsa")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	payload := []byte(`{"test": "data"}`)
	compact, err := signer.Sign(payload)
	require.NoError(t, err)
	assert.NotEmpty(t, compact)

	// Verify with the certificate
	cert, err := parseCertFile(certPath)
	require.NoError(t, err)
	verifier := NewCertVerifier(cert)
	verified, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, payload, verified)
}

func TestFileSigner_EC256_SignVerify(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "ec256")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	payload := []byte(`{"hello": "world"}`)
	compact, err := signer.Sign(payload)
	require.NoError(t, err)

	cert, err := parseCertFile(certPath)
	require.NoError(t, err)
	verifier := NewCertVerifier(cert)
	verified, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, payload, verified)
}

func TestFileSigner_EC384_SignVerify(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "ec384")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	payload := []byte(`test payload`)
	compact, err := signer.Sign(payload)
	require.NoError(t, err)

	cert, err := parseCertFile(certPath)
	require.NoError(t, err)
	verifier := NewCertVerifier(cert)
	verified, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, payload, verified)
}

func TestKeyVerifier_WrongKey(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "ec256")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	compact, err := signer.Sign([]byte("data"))
	require.NoError(t, err)

	// Verify with a different key
	wrongKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	verifier := NewKeyVerifier(&wrongKey.PublicKey)
	_, err = verifier.Verify(compact)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no trusted key matched")
}

func TestNewFileSigner_MissingKeyFile(t *testing.T) {
	dir := t.TempDir()
	certPath, _ := generateTestCertAndKey(t, dir, "rsa")
	_, err := NewFileSigner(certPath, "/nonexistent/key.pem")
	assert.Error(t, err)
}

func TestNewFileSigner_MissingCertFile(t *testing.T) {
	dir := t.TempDir()
	_, keyPath := generateTestCertAndKey(t, dir, "rsa")
	_, err := NewFileSigner("/nonexistent/cert.pem", keyPath)
	assert.Error(t, err)
}

func TestKeyVerifier_InvalidJWS(t *testing.T) {
	verifier := NewKeyVerifier()
	_, err := verifier.Verify("not.a.jws")
	assert.Error(t, err)
}

// parseCertFile is a test helper to load a cert from a PEM file.
func parseCertFile(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	certs, err := parseCertificates(data)
	if err != nil {
		return nil, err
	}
	return certs[0], nil
}

func TestFileSigner_JAdES_Headers(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "ec256")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	payload := []byte(`{"test": "jades"}`)
	compact, err := signer.Sign(payload)
	require.NoError(t, err)

	// Parse the JWS to inspect headers
	parsed, err := jose.ParseSigned(compact, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	require.Len(t, parsed.Signatures, 1)

	headers := parsed.Signatures[0].Protected
	// sigT must be present and be a valid RFC 3339 timestamp
	sigT, ok := headers.ExtraHeaders["sigT"]
	assert.True(t, ok, "sigT header must be present for JAdES-B-B")
	sigTStr, ok := sigT.(string)
	assert.True(t, ok, "sigT must be a string")
	_, err = time.Parse(time.RFC3339, sigTStr)
	assert.NoError(t, err, "sigT must be valid RFC 3339")

	// x5t#S256 must be present
	thumbprint, ok := headers.ExtraHeaders["x5t#S256"]
	assert.True(t, ok, "x5t#S256 header must be present for JAdES-B-B")
	assert.NotEmpty(t, thumbprint)

	// Verify the JWS still verifies correctly
	cert, err := parseCertFile(certPath)
	require.NoError(t, err)
	verifier := NewCertVerifier(cert)
	verified, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, payload, verified)
}

func TestFileSigner_JAdES_Disabled(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "rsa")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)
	signer.SetJAdES(false)

	payload := []byte(`{"test": "plain"}`)
	compact, err := signer.Sign(payload)
	require.NoError(t, err)

	// Parse the JWS to inspect headers
	parsed, err := jose.ParseSigned(compact, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	require.Len(t, parsed.Signatures, 1)

	headers := parsed.Signatures[0].Protected
	_, hasSigT := headers.ExtraHeaders["sigT"]
	assert.False(t, hasSigT, "sigT must not be present when JAdES is disabled")
	_, hasThumb := headers.ExtraHeaders["x5t#S256"]
	assert.False(t, hasThumb, "x5t#S256 must not be present when JAdES is disabled")

	// Verify still works
	cert, err := parseCertFile(certPath)
	require.NoError(t, err)
	verifier := NewCertVerifier(cert)
	verified, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, payload, verified)
}

func TestFileSigner_JAdES_Thumbprint_Matches_Cert(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "ec256")

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	compact, err := signer.Sign([]byte("test"))
	require.NoError(t, err)

	// Parse the JWS
	parsed, err := jose.ParseSigned(compact, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)

	// Compute expected thumbprint from cert
	cert, err := parseCertFile(certPath)
	require.NoError(t, err)
	expectedThumbprint := sha256.Sum256(cert.Raw)
	expectedB64 := base64.RawURLEncoding.EncodeToString(expectedThumbprint[:])

	actual := parsed.Signatures[0].Protected.ExtraHeaders["x5t#S256"]
	assert.Equal(t, expectedB64, actual, "x5t#S256 must match SHA-256 of signing cert DER")
}
