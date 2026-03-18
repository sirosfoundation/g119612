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
