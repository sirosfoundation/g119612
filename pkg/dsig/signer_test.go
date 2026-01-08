package dsig

import (
	"testing"
)

func TestGetSigningMethodName(t *testing.T) {
	result := GetSigningMethodName()
	expected := "rsa-sha256"

	if result != expected {
		t.Errorf("GetSigningMethodName() = %v, want %v", result, expected)
	}
}
