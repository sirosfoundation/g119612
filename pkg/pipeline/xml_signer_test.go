package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateXMLSigner_NoArgs(t *testing.T) {
	signer, err := createXMLSigner(nil)
	require.NoError(t, err)
	assert.Nil(t, signer, "no args should return nil signer")
}

func TestCreateXMLSigner_JadesOnlyFiltered(t *testing.T) {
	// Only jades:false should be filtered out, leaving no signing args
	signer, err := createXMLSigner([]string{"jades:false"})
	require.NoError(t, err)
	assert.Nil(t, signer, "jades:false alone should return nil signer")
}

func TestCreateXMLSigner_InsufficientArgs(t *testing.T) {
	// Single non-pkcs11 arg → not enough for file-based signing
	_, err := createXMLSigner([]string{"/path/to/cert.pem"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires cert and key paths")
}

func TestCreateXMLSigner_InvalidPKCS11(t *testing.T) {
	_, err := createXMLSigner([]string{"pkcs11:invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PKCS#11")
}

func TestFilterSignerArgs(t *testing.T) {
	args := []string{"/cert.pem", "/key.pem", "json-only", "xml-only"}
	filtered := filterSignerArgs(args)
	assert.Equal(t, []string{"/cert.pem", "/key.pem"}, filtered)
}

func TestFilterLoteXMLArgs(t *testing.T) {
	args := []string{"/cert.pem", "/key.pem", "xml", "jades:false"}
	filtered := filterLoteXMLArgs(args)
	assert.Equal(t, []string{"/cert.pem", "/key.pem"}, filtered)
}

func TestParseLotlFormat(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		f := parseLotlFormat(nil)
		assert.True(t, f.json)
		assert.True(t, f.xml)
	})

	t.Run("json-only", func(t *testing.T) {
		f := parseLotlFormat([]string{"json-only"})
		assert.True(t, f.json)
		assert.False(t, f.xml)
	})

	t.Run("xml-only", func(t *testing.T) {
		f := parseLotlFormat([]string{"xml-only"})
		assert.False(t, f.json)
		assert.True(t, f.xml)
	})
}

func TestParseLoteXMLFlags(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		wantXML, xmlOnly := parseLoteXMLFlags(nil)
		assert.False(t, wantXML)
		assert.False(t, xmlOnly)
	})

	t.Run("xml", func(t *testing.T) {
		wantXML, xmlOnly := parseLoteXMLFlags([]string{"xml"})
		assert.True(t, wantXML)
		assert.False(t, xmlOnly)
	})

	t.Run("xml-only", func(t *testing.T) {
		wantXML, xmlOnly := parseLoteXMLFlags([]string{"xml-only"})
		assert.True(t, wantXML)
		assert.True(t, xmlOnly)
	})
}
