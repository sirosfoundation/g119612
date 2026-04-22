package pipeline

import (
	"fmt"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/dsig"
)

// createXMLSigner creates an XML-DSIG signer from the publish arguments.
// Returns nil signer if no signing args provided.
func createXMLSigner(args []string) (dsig.XMLSigner, error) {
	if len(args) == 0 {
		return nil, nil
	}

	// Filter out jades:false and format flags (those are for JWS signer)
	var filteredArgs []string
	for _, arg := range args {
		switch {
		case arg == "jades:false":
			continue
		default:
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	if len(args) == 0 {
		return nil, nil
	}

	// PKCS#11: first arg starts with "pkcs11:"
	if strings.HasPrefix(args[0], "pkcs11:") {
		config := dsig.ExtractPKCS11Config(args[0])
		if config == nil {
			return nil, fmt.Errorf("invalid PKCS#11 URI for XML signing: %s", args[0])
		}
		keyLabel := "default-key"
		certLabel := "default-cert"
		if len(args) >= 2 {
			keyLabel = args[1]
		}
		if len(args) >= 3 {
			certLabel = args[2]
		}
		signer := dsig.NewPKCS11Signer(config, keyLabel, certLabel)
		return signer, nil
	}

	// File-based: need cert and key paths
	if len(args) >= 2 {
		return dsig.NewFileSigner(args[0], args[1]), nil
	}

	return nil, fmt.Errorf("XML signing requires cert and key paths, or a pkcs11: URI")
}
