package pipeline

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"gopkg.in/yaml.v3"
)

// MultiLangName represents a name in a specific language
type MultiLangName struct {
	Language string `yaml:"language"`
	Value    string `yaml:"value"`
}

// Address represents a postal and electronic address
type Address struct {
	Postal struct {
		StreetAddress   string `yaml:"streetAddress"`
		Locality        string `yaml:"locality"`
		StateOrProvince string `yaml:"stateOrProvince,omitempty"`
		PostalCode      string `yaml:"postalCode,omitempty"`
		CountryName     string `yaml:"countryName"`
	} `yaml:"postal"`
	Electronic []string `yaml:"electronic,omitempty"`
}

// ProviderMetadata represents the YAML structure for a provider's metadata
type ProviderMetadata struct {
	Names          []MultiLangName `yaml:"names"` // At least one name required
	Address        *Address        `yaml:"address,omitempty"`
	TradeName      []MultiLangName `yaml:"tradeName,omitempty"`
	InformationURI []MultiLangName `yaml:"informationURI,omitempty"`
}

// CertificateMetadata represents the YAML structure for a certificate's metadata
type CertificateMetadata struct {
	ServiceNames     []MultiLangName `yaml:"serviceNames"` // At least one name required
	ServiceType      string          `yaml:"serviceType"`  // URI identifying the service type
	Status           string          `yaml:"status"`       // Must be a valid TSL status URI
	ServiceDigitalID *struct {
		DigitalIDs []string `yaml:"digitalIds,omitempty"` // Additional digital IDs beyond the certificate
	} `yaml:"serviceDigitalId,omitempty"`
}

// SchemeMetadata represents the YAML structure for the TSL scheme metadata
type SchemeMetadata struct {
	OperatorNames  []MultiLangName `yaml:"operatorNames"`            // At least one name required
	Type           string          `yaml:"type"`                     // URI identifying the TSL type
	SequenceNumber int             `yaml:"sequenceNumber,omitempty"` // TSL sequence number
}

// loadSchemeMetadata loads and parses the scheme metadata from the scheme.yaml file.
// This function reads the top-level TSL configuration including operator names,
// TSL type URI, and sequence number.
//
// The scheme.yaml file must contain:
//   - operatorNames: At least one operator name with language and value
//   - type: A valid TSL type URI (e.g., http://uri.etsi.org/TrstSvc/TrustedList/TSLType/...)
//   - sequenceNumber: Optional TSL sequence number (defaults to 1 if not provided)
//
// Parameters:
//   - rootDir: Absolute path to the root directory containing scheme.yaml
//
// Returns:
//   - *SchemeMetadata: Parsed scheme metadata structure
//   - error: If the file cannot be read, is not valid YAML, or missing required fields
//
// Example scheme.yaml:
//
//	operatorNames:
//	  - language: en
//	    value: "Trust List Operator"
//	type: "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUlistofthelists"
//	sequenceNumber: 1
func loadSchemeMetadata(rootDir string) (*SchemeMetadata, error) {
	metadataPath := filepath.Join(rootDir, "scheme.yaml")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scheme metadata from %s: %w", metadataPath, err)
	}

	var metadata SchemeMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse scheme metadata from %s: %w", metadataPath, err)
	}

	if len(metadata.OperatorNames) == 0 {
		return nil, fmt.Errorf("scheme metadata must include at least one operator name")
	}

	if metadata.Type == "" {
		return nil, fmt.Errorf("scheme metadata must include a type URI")
	}

	return &metadata, nil
}

// loadProviderMetadata loads and parses the provider metadata from provider.yaml.
// This function reads provider-specific information such as names, addresses,
// trade names, and information URIs in multiple languages.
//
// The provider.yaml file must contain:
//   - names: At least one provider name with language and value
//   - address: Optional postal and electronic addresses
//   - tradeName: Optional trade names in multiple languages
//   - informationURI: Optional information URIs in multiple languages
//
// Parameters:
//   - providerDir: Absolute path to the provider directory containing provider.yaml
//
// Returns:
//   - *ProviderMetadata: Parsed provider metadata structure
//   - error: If the file cannot be read, is not valid YAML, or missing required fields
//
// Example provider.yaml:
//
//	names:
//	  - language: en
//	    value: "Example Trust Service Provider"
//	address:
//	  postal:
//	    streetAddress: "Example Street 123"
//	    locality: "Example City"
//	    postalCode: "12345"
//	    countryName: "SE"
//	  electronic:
//	    - "https://example.com"
//	    - "mailto:contact@example.com"
func loadProviderMetadata(providerDir string) (*ProviderMetadata, error) {
	metadataPath := filepath.Join(providerDir, "provider.yaml")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider metadata from %s: %w", metadataPath, err)
	}

	var metadata ProviderMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse provider metadata from %s: %w", metadataPath, err)
	}

	if len(metadata.Names) == 0 {
		return nil, fmt.Errorf("provider metadata must include at least one name")
	}

	return &metadata, nil
}

// addProviderCertificates processes certificate files in a provider directory and adds them to the TSP.
// For each .pem certificate file, it looks for a corresponding .yaml metadata file
// with the same base name. The function handles both the certificate data and its
// service metadata to populate the TSP's service list.
//
// For each certificate pair (example.pem + example.yaml):
//  1. Reads and parses the X.509 certificate from the .pem file
//  2. Loads the service metadata from the .yaml file
//  3. Creates a TSP service entry with the certificate and metadata
//  4. Adds the service to the provider's service list
//
// Parameters:
//   - providerDir: Absolute path to the provider directory containing .pem and .yaml files
//   - provider: TSP structure to add the certificates and services to
//
// Returns:
//   - error: If any certificate or metadata file cannot be read or parsed
//
// Expected files:
//   - *.pem: X.509 certificates in PEM format
//   - *.yaml: Matching metadata files for each certificate
//
// Example cert.yaml:
//
//	serviceNames:
//	  - language: en
//	    value: "Example Certificate Service"
//	serviceType: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC"
//	status: "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/"
func addProviderCertificates(providerDir string, provider *etsi119612.TSPType) error {
	entries, err := os.ReadDir(providerDir)
	if err != nil {
		return fmt.Errorf("failed to read provider directory %s: %w", providerDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pem") {
			continue
		}

		certPath := filepath.Join(providerDir, entry.Name())
		metadataPath := certPath[:len(certPath)-4] + ".yaml" // replace .pem with .yaml

		// Load certificate metadata
		metadataBytes, err := os.ReadFile(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to read certificate metadata from %s: %w", metadataPath, err)
		}

		var metadata CertificateMetadata
		if err := yaml.Unmarshal(metadataBytes, &metadata); err != nil {
			return fmt.Errorf("failed to parse certificate metadata from %s: %w", metadataPath, err)
		}

		if len(metadata.ServiceNames) == 0 {
			return fmt.Errorf("certificate metadata must include at least one service name")
		}

		// Load certificate
		certBytes, err := os.ReadFile(certPath)
		if err != nil {
			return fmt.Errorf("failed to read certificate from %s: %w", certPath, err)
		}

		// Try to parse the certificate to ensure it's valid
		_, err = x509.ParseCertificate(certBytes)
		if err != nil {
			return fmt.Errorf("failed to decode invalid certificate data in %s: %w", certPath, err)
		}

		// Create service names
		serviceNames := make([]*etsi119612.MultiLangNormStringType, len(metadata.ServiceNames))
		for i, name := range metadata.ServiceNames {
			serviceNames[i] = &etsi119612.MultiLangNormStringType{
				XmlLangAttr: func() *etsi119612.Lang {
					l := etsi119612.Lang(name.Language)
					return &l
				}(),
				NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
					s := etsi119612.NonEmptyNormalizedString(name.Value)
					return &s
				}(),
			}
		}

		// Create digital IDs - certificate bytes have been validated above
		digitalIds := []*etsi119612.DigitalIdentityType{
			{
				X509Certificate: base64.StdEncoding.EncodeToString(certBytes),
			},
		}

		if metadata.ServiceDigitalID != nil {
			for _, id := range metadata.ServiceDigitalID.DigitalIDs {
				digitalIds = append(digitalIds, &etsi119612.DigitalIdentityType{
					X509Certificate: id,
				})
			}
		}

		// Create service entry
		service := &etsi119612.TSPServiceType{
			TslServiceInformation: &etsi119612.TSPServiceInformationType{
				TslServiceTypeIdentifier: metadata.ServiceType,
				TslServiceStatus:         metadata.Status,
				ServiceName: &etsi119612.InternationalNamesType{
					Name: serviceNames,
				},
				TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
					DigitalId: digitalIds,
				},
			},
		}

		provider.TslTSPServices.TslTSPService = append(
			provider.TslTSPServices.TslTSPService,
			service,
		)
	}

	return nil
}

// GenerateTSL is a pipeline step that generates a Trust Service List (TSL) from a structured directory.
// It implements generation of ETSI TS 119612 compliant TSLs by reading metadata and certificates
// from a hierarchical directory structure.
//
// Directory Structure:
//
//	root/
//	  ├── scheme.yaml      # TSL scheme metadata
//	  └── providers/       # Directory containing all providers
//	      └── provider1/   # One directory per provider
//	          ├── provider.yaml  # Provider metadata
//	          ├── cert1.pem      # Certificate files
//	          └── cert1.yaml     # Certificate metadata
//
// File Formats:
//
//	scheme.yaml:
//	  operatorNames:       # List of operator names in different languages
//	    - language: en
//	      value: "Trust List Operator"
//	  type: "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/..."  # TSL type URI
//	  sequenceNumber: 1    # TSL sequence number
//
//	provider.yaml:
//	  names:              # List of provider names in different languages
//	    - language: en
//	      value: "Example Provider"
//	  address:            # Provider's address information
//	    postal:
//	      streetAddress: "Example Street 123"
//	      locality: "Example City"
//	      postalCode: "12345"
//	      countryName: "SE"
//	    electronic:        # List of electronic addresses
//	      - "https://example.com"
//	      - "mailto:contact@example.com"
//	  tradeName:          # Optional trade names in different languages
//	    - language: en
//	      value: "Example Corp"
//	  informationURI:     # Optional information URIs in different languages
//	    - language: en
//	      value: "https://example.com/info"
//
//	cert.yaml (matching .pem file):
//	  serviceNames:        # List of service names in different languages
//	    - language: en
//	      value: "Example Service"
//	  serviceType: "http://uri.etsi.org/TrstSvc/Svctype/..."  # Service type URI
//	  status: "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/..."  # Status URI
//	  serviceDigitalId:    # Optional additional digital IDs
//	    digitalIds:
//	      - "base64 encoded cert..."
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: String slice where args[0] must be the path to the root directory
//
// Returns:
//   - *Context: Updated context with the generated TSL added to ctx.TSLs
//   - error: Non-nil if any error occurs during generation
//
// The function generates a TSL by:
// 1. Loading scheme metadata from scheme.yaml
// 2. Creating the base TSL structure with scheme information
// 3. Iterating through provider directories in the providers/ subdirectory
// 4. For each provider:
//   - Loading provider metadata and creating TSP entries
//   - Processing all certificate files (.pem) and their metadata (.yaml)
//   - Adding all services and certificates to the provider entry
//
// 5. Adding the complete TSL to the pipeline context
//   - rootDir: path to the root directory containing scheme.yaml and providers directory
func GenerateTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("GenerateTSL requires 1 argument: path to root directory")
	}

	rootDir := args[0]
	providersDir := filepath.Join(rootDir, "providers")
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers directory %s: %w", providersDir, err)
	}

	// Load scheme metadata
	schemeMetadata, err := loadSchemeMetadata(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load scheme metadata: %w", err)
	}

	// Create operator names for the TSL
	operatorNames := make([]*etsi119612.MultiLangNormStringType, len(schemeMetadata.OperatorNames))
	for i, name := range schemeMetadata.OperatorNames {
		operatorNames[i] = &etsi119612.MultiLangNormStringType{
			XmlLangAttr: func() *etsi119612.Lang {
				l := etsi119612.Lang(name.Language)
				return &l
			}(),
			NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
				s := etsi119612.NonEmptyNormalizedString(name.Value)
				return &s
			}(),
		}
	}

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TSLVersionIdentifier: int(schemeMetadata.SequenceNumber),
				TslTSLType:           schemeMetadata.Type,
				TslSchemeOperatorName: &etsi119612.InternationalNamesType{
					Name: operatorNames,
				},
			},
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{},
			},
		},
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		providerDir := filepath.Join(providersDir, entry.Name())
		providerMetadata, err := loadProviderMetadata(providerDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load provider metadata from %s: %w", providerDir, err)
		}

		// Create provider names
		providerNames := make([]*etsi119612.MultiLangNormStringType, len(providerMetadata.Names))
		for i, name := range providerMetadata.Names {
			providerNames[i] = &etsi119612.MultiLangNormStringType{
				XmlLangAttr: func() *etsi119612.Lang {
					l := etsi119612.Lang(name.Language)
					return &l
				}(),
				NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
					s := etsi119612.NonEmptyNormalizedString(name.Value)
					return &s
				}(),
			}
		}

		provider := &etsi119612.TSPType{
			TslTSPInformation: &etsi119612.TSPInformationType{
				TSPName: &etsi119612.InternationalNamesType{
					Name: providerNames,
				},
			},
			TslTSPServices: &etsi119612.TSPServicesListType{
				TslTSPService: []*etsi119612.TSPServiceType{},
			},
		}

		// Add provider address if present
		if providerMetadata.Address != nil {
			provider.TslTSPInformation.TSPAddress = &etsi119612.AddressType{
				TslPostalAddresses: &etsi119612.PostalAddressListType{
					TslPostalAddress: []*etsi119612.PostalAddressType{
						{
							XmlLangAttr:     func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
							StreetAddress:   providerMetadata.Address.Postal.StreetAddress,
							Locality:        providerMetadata.Address.Postal.Locality,
							StateOrProvince: providerMetadata.Address.Postal.StateOrProvince,
							PostalCode:      providerMetadata.Address.Postal.PostalCode,
							CountryName:     providerMetadata.Address.Postal.CountryName,
						},
					},
				},
			}

			if len(providerMetadata.Address.Electronic) > 0 {
				electronic := make([]*etsi119612.NonEmptyMultiLangURIType, len(providerMetadata.Address.Electronic))
				for i, uri := range providerMetadata.Address.Electronic {
					electronic[i] = &etsi119612.NonEmptyMultiLangURIType{
						XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
						Value:       uri,
					}
				}
				provider.TslTSPInformation.TSPAddress.TslElectronicAddress = &etsi119612.ElectronicAddressType{
					URI: electronic,
				}
			}
		}

		err = addProviderCertificates(providerDir, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to add certificates for provider %s: %w", entry.Name(), err)
		}

		tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider = append(
			tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider,
			provider,
		)
	}

	ctx.EnsureTSLStack().TSLs.Push(tsl)

	return ctx, nil
}

// LoadTSL is a pipeline step that loads a Trust Service List (TSL) from a file or URL.
// This function supports loading TSLs from both local files and remote HTTP(S) URLs,
// and will also load any referenced TSLs based on the MaxDereferenceDepth setting.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: String slice where:
//   - args[0] must be the URL or file path to the TSL
//
// Returns:
//   - *Context: Updated context with the loaded TSL and referenced TSLs added to ctx.TSLs
//   - error: Non-nil if the TSL cannot be loaded or parsed
//
// URL handling:
//   - HTTP(S) URLs are used as-is
//   - Local paths are converted to file:// URLs
//   - The TSL is fetched and parsed using etsi119612.FetchTSLWithReferencesAndOptions
//
// The function uses the fetch options (UserAgent, Timeout, MaxDereferenceDepth) that
// were previously set using SetFetchOptions. If not set, default values will be used.
//
// The loaded TSL and all referenced TSLs (according to MaxDereferenceDepth) are pushed
// onto the context's TSL stack, with the root TSL on top. If the stack doesn't exist,
// a new one is created. Multiple calls to LoadTSL will result in multiple TSLs being
// available in the context.
//
// Example usage in pipeline configuration:
//   - set-fetch-options:
//   - user-agent:MyCustomUserAgent/1.0
//   - timeout:60s
//   - max-depth:3
//   - load: [http://example.com/tsl.xml]
//   - load: [/path/to/local/tsl.xml]
