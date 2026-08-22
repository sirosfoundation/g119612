// Go 1.23 made crypto/x509.ParseCertificate reject certificates whose
// DER-encoded serial number is negative. RFC 5280 requires a positive
// integer, but encoders that omit the leading zero padding byte are
// widespread: reader/verifier CA certificates from BouncyCastle-based
// stacks -- notably the Android/multipaz ecosystem -- routinely trip this,
// and Go is unusually strict in rejecting them outright.
//
// A trust list only needs each certificate to be well-formed enough to be
// carried and matched against a presented chain; the sign of the serial
// carries no security meaning in that role. Refusing to parse makes such
// trust lists impossible to publish at all, so restore the pre-1.23
// behaviour for the whole tsl-tool binary.
//
// This affects only the tsl-tool command. Callers using g119612 as a
// library keep the Go default and can opt in with their own //go:debug
// line or a GODEBUG=x509negativeserial=1 environment variable.
//
//go:debug x509negativeserial=1

package main
