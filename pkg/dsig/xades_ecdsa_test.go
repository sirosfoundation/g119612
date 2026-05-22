package dsig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"testing"
	"time"
)

func TestEnsureP1363Signature_ECDSA_P256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	digest := sha256.Sum256([]byte("test data"))
	derSig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatalf("SignASN1: %v", err)
	}

	p1363, err := ensureP1363Signature(derSig, &key.PublicKey)
	if err != nil {
		t.Fatalf("ensureP1363Signature: %v", err)
	}

	if len(p1363) != 64 {
		t.Fatalf("P1363 signature length: got %d, want 64", len(p1363))
	}

	// Verify the P1363 signature by extracting r and s
	r := new(big.Int).SetBytes(p1363[:32])
	s := new(big.Int).SetBytes(p1363[32:])
	if !ecdsa.Verify(&key.PublicKey, digest[:], r, s) {
		t.Fatal("P1363 signature does not verify")
	}
}

func TestEnsureP1363Signature_RSA_Passthrough(t *testing.T) {
	raw := []byte("not-an-ecdsa-key")
	// For RSA (nil ECDSA key), should pass through unchanged
	type fakeRSAKey struct{}
	result, err := ensureP1363Signature(raw, &fakeRSAKey{})
	if err != nil {
		t.Fatalf("ensureP1363Signature with non-ECDSA key: %v", err)
	}
	if string(result) != string(raw) {
		t.Fatal("non-ECDSA signature should pass through unchanged")
	}
}

func TestP1363ToDER_Roundtrip(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	digest := sha256.Sum256([]byte("roundtrip test"))
	derSig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatalf("SignASN1: %v", err)
	}

	// DER → P1363
	p1363, err := ensureP1363Signature(derSig, &key.PublicKey)
	if err != nil {
		t.Fatalf("ensureP1363Signature: %v", err)
	}

	// P1363 → DER
	derAgain, err := p1363ToDER(p1363, elliptic.P256())
	if err != nil {
		t.Fatalf("p1363ToDER: %v", err)
	}

	// Verify the round-tripped DER signature
	var sig ecdsaDERSignature
	if _, err := asn1.Unmarshal(derAgain, &sig); err != nil {
		t.Fatalf("Unmarshal round-tripped DER: %v", err)
	}
	if !ecdsa.Verify(&key.PublicKey, digest[:], sig.R, sig.S) {
		t.Fatal("round-tripped DER signature does not verify")
	}
}

func generateECDSACert(t *testing.T) (*ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "XAdES ECDSA Test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}

	return key, cert
}

func TestSignXMLWithXAdES_ECDSA(t *testing.T) {
	key, cert := generateECDSACert(t)

	xmlData := []byte(`<Root xmlns="urn:test"><Value>hello</Value></Root>`)

	signed, err := SignXMLWithXAdES(xmlData, key, cert)
	if err != nil {
		t.Fatalf("SignXMLWithXAdES: %v", err)
	}

	if len(signed) == 0 {
		t.Fatal("signed output is empty")
	}

	// Verify the signature using our verifier
	certs := []*x509.Certificate{cert}
	if _, err := VerifyXMLSignature(signed, certs); err != nil {
		t.Fatalf("VerifyXMLSignature failed: %v", err)
	}
}
