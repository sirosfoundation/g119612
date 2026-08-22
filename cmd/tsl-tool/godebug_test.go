package main

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

// TestNegativeSerialIsAccepted guards the //go:debug x509negativeserial=1
// directive in godebug.go. Go 1.23 began rejecting certificates whose DER
// serial number encodes as a negative ASN.1 INTEGER, which is common in
// BouncyCastle-derived reader CA certificates that trust lists must be able
// to carry. Without the directive this fails with "x509: negative serial
// number" and those lists cannot be published at all.
//
// The fixture is a self-signed certificate whose serial has the top bit set
// with no padding byte. Its signature does not cover the patched bytes, which
// does not matter here: ParseCertificate does not verify signatures, and the
// parse is the behaviour under test.
func TestNegativeSerialIsAccepted(t *testing.T) {
	data, err := os.ReadFile("testdata/negative-serial-ca.pem")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("fixture is not valid PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parsing certificate with negative serial: %v "+
			"(is //go:debug x509negativeserial=1 still present in godebug.go?)", err)
	}
	if cert.SerialNumber.Sign() >= 0 {
		t.Fatalf("fixture no longer has a negative serial: %s", cert.SerialNumber)
	}
}
