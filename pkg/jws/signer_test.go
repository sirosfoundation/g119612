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

func TestNewCertFileVerifier(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKey(t, dir, "ec256")

	// Create a signed JWS
	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	payload := []byte(`{"verifier_test": true}`)
	compact, err := signer.Sign(payload)
	require.NoError(t, err)

	// Verify using NewCertFileVerifier
	verifier, err := NewCertFileVerifier(certPath)
	require.NoError(t, err)

	verified, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, payload, verified)
}

func TestFileSigner_EC521_SignVerify(t *testing.T) {
	dir := t.TempDir()

	// Generate EC P-521 key and cert
	ecKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-ec521"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &ecKey.PublicKey, ecKey)
	require.NoError(t, err)

	certPath := filepath.Join(dir, "cert.pem")
	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	certFile.Close()

	keyPath := filepath.Join(dir, "key.pem")
	keyFile, err := os.Create(keyPath)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(keyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
	keyFile.Close()

	signer, err := NewFileSigner(certPath, keyPath)
	require.NoError(t, err)

	payload := []byte(`{"curve": "P-521"}`)
	compact, err := signer.Sign(payload)
	require.NoError(t, err)

	verifier, err := NewCertFileVerifier(certPath)
	require.NoError(t, err)

	verified, err := verifier.Verify(compact)
	require.NoError(t, err)
	assert.Equal(t, payload, verified)
}

func TestParsePrivateKey_RSA_PKCS1(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Marshal as PKCS#1 (RSA PRIVATE KEY)
	keyDER := x509.MarshalPKCS1PrivateKey(rsaKey)
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER})

	parsed, err := parsePrivateKey(pemBlock)
	require.NoError(t, err)
	assert.IsType(t, &rsa.PrivateKey{}, parsed)
}

func TestParsePrivateKey_EC(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Marshal as SEC 1 (EC PRIVATE KEY)
	keyDER, err := x509.MarshalECPrivateKey(ecKey)
	require.NoError(t, err)
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	parsed, err := parsePrivateKey(pemBlock)
	require.NoError(t, err)
	assert.IsType(t, &ecdsa.PrivateKey{}, parsed)
}

func TestParsePrivateKey_NoPEM(t *testing.T) {
	_, err := parsePrivateKey([]byte("not pem data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no PEM block found")
}

func TestParsePrivateKey_UnsupportedType(t *testing.T) {
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "UNKNOWN KEY TYPE", Bytes: []byte("data")})
	_, err := parsePrivateKey(pemBlock)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported PEM block type")
}

func TestAlgorithmForKey_UnsupportedCurve(t *testing.T) {
	// Create a mock EC key with an unsupported curve (use P224 which is not in the switch)
	ecKey, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	require.NoError(t, err)

	_, err = algorithmForKey(ecKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported EC curve")
}

func TestAlgorithmForKey_UnsupportedKeyType(t *testing.T) {
	_, err := algorithmForKey("not a key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key type")
}

func TestNewFileSigner_InvalidKeyPEM(t *testing.T) {
	dir := t.TempDir()
	certPath, _ := generateTestCertAndKey(t, dir, "rsa")

	// Create an invalid key file
	invalidKeyPath := filepath.Join(dir, "invalid_key.pem")
	err := os.WriteFile(invalidKeyPath, []byte("not a valid PEM key"), 0644)
	require.NoError(t, err)

	_, err = NewFileSigner(certPath, invalidKeyPath)
	assert.Error(t, err)
}

func TestNewFileSigner_InvalidCertPEM(t *testing.T) {
	dir := t.TempDir()
	_, keyPath := generateTestCertAndKey(t, dir, "rsa")

	// Create an invalid cert file
	invalidCertPath := filepath.Join(dir, "invalid_cert.pem")
	err := os.WriteFile(invalidCertPath, []byte("not a valid PEM cert"), 0644)
	require.NoError(t, err)

	_, err = NewFileSigner(invalidCertPath, keyPath)
	assert.Error(t, err)
}

func TestParseCertificates_InvalidCert(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid cert data")})
	_, err := parseCertificates(pemData)
	assert.Error(t, err)
}

func TestCertsToX5C(t *testing.T) {
	dir := t.TempDir()
	certPath, _ := generateTestCertAndKey(t, dir, "rsa")

	cert, err := parseCertFile(certPath)
	require.NoError(t, err)

	x5c := certsToX5C([]*x509.Certificate{cert})
	assert.Len(t, x5c, 1)
	assert.Equal(t, cert.Raw, x5c[0])
}

func TestMultipleCertificatesInChain(t *testing.T) {
	dir := t.TempDir()

	// Generate root CA
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	require.NoError(t, err)
	rootCert, err := x509.ParseCertificate(rootDER)
	require.NoError(t, err)

	// Generate leaf cert
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "leaf"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, rootCert, &leafKey.PublicKey, rootKey)
	require.NoError(t, err)

	// Write chain PEM file (leaf + root)
	chainPath := filepath.Join(dir, "chain.pem")
	chainFile, err := os.Create(chainPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(chainFile, &pem.Block{Type: "CERTIFICATE", Bytes: leafDER}))
	require.NoError(t, pem.Encode(chainFile, &pem.Block{Type: "CERTIFICATE", Bytes: rootDER}))
	chainFile.Close()

	// Parse the chain
	data, err := os.ReadFile(chainPath)
	require.NoError(t, err)
	certs, err := parseCertificates(data)
	require.NoError(t, err)
	assert.Len(t, certs, 2)
	assert.Equal(t, "leaf", certs[0].Subject.CommonName)
	assert.Equal(t, "root", certs[1].Subject.CommonName)
}
