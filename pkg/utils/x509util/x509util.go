// Package x509util provides utility functions for parsing X.509 certificates
// from various formats commonly used in authentication protocols.
package x509util

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// ParseX5CFromArray parses X.509 certificates from an array of base64-encoded DER certificates.
// This is used when the resource.type is "x5c" in the AuthZEN Trust Registry Profile.
// Each element in the array should be a base64-encoded X.509 DER certificate.
func ParseX5CFromArray(key []interface{}) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	if len(key) == 0 {
		return nil, fmt.Errorf("resource.key is empty")
	}

	for i, item := range key {
		str, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("resource.key[%d] is not a string", i)
		}
		der, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode resource.key[%d]: %v", i, err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate from resource.key[%d]: %v", i, err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// ParseX5CFromJWK parses X.509 certificates from a JWK (JSON Web Key) structure.
// This is used when resource.type is "jwk" in the AuthZEN Trust Registry Profile.
// The JWK may contain an "x5c" claim which is an array of base64-encoded DER certificates.
// The resource.key array should contain a single JWK object as a map[string]interface{}.
func ParseX5CFromJWK(key []interface{}) ([]*x509.Certificate, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("resource.key is empty")
	}

	// The first element should be a JWK object
	jwk, ok := key[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resource.key[0] is not a JWK object (map)")
	}

	// Extract x5c claim from JWK
	x5cVal, ok := jwk["x5c"]
	if !ok {
		return nil, fmt.Errorf("JWK does not contain x5c claim")
	}

	x5cList, ok := x5cVal.([]interface{})
	if !ok {
		return nil, fmt.Errorf("JWK x5c claim is not an array")
	}

	var certs []*x509.Certificate
	for i, item := range x5cList {
		str, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("JWK x5c[%d] is not a string", i)
		}
		der, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode JWK x5c[%d]: %v", i, err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate from JWK x5c[%d]: %v", i, err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}
