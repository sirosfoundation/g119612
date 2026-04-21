package etsi119602

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeStatusURI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "http no trailing slash",
			input:    "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
			expected: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		},
		{
			name:     "https with trailing slash",
			input:    "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/",
			expected: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		},
		{
			name:     "https no trailing slash",
			input:    "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
			expected: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		},
		{
			name:     "http with trailing slash",
			input:    "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/",
			expected: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		},
		{
			name:     "withdrawn https",
			input:    "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/withdrawn/",
			expected: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/withdrawn",
		},
		{
			name:     "non-ETSI URI unchanged",
			input:    "https://example.com/status/active",
			expected: "https://example.com/status/active",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace trimmed",
			input:    "  http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted  ",
			expected: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeStatusURI(tt.input))
		})
	}
}

func TestStatusEquals(t *testing.T) {
	assert.True(t, StatusEquals(
		"http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		"https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/",
	))

	assert.True(t, StatusEquals(
		"https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		"http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/",
	))

	assert.False(t, StatusEquals(
		"http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
		"http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/withdrawn",
	))
}

func TestNormalizeServiceTypeURI(t *testing.T) {
	assert.Equal(t,
		"http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
		NormalizeServiceTypeURI("https://uri.etsi.org/TrstSvc/Svctype/CA/QC/"),
	)
}
