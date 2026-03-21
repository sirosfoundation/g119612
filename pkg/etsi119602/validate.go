package etsi119602

import (
	"fmt"
	"strings"
)

// validDigitalIdentityTypes are the supported DigitalIdentity type values.
var validDigitalIdentityTypes = map[string]bool{
	"jwk":               true,
	"x509":              true,
	"did":               true,
	"x509_subject_name": true,
}

// Validate checks the LoTE document for structural correctness.
// It ensures required fields are present and values are well-formed.
func (l *ListOfTrustedEntities) Validate() error {
	if l.Version == "" {
		return fmt.Errorf("version is required")
	}
	if err := l.SchemeInformation.validate(); err != nil {
		return fmt.Errorf("schemeInformation: %w", err)
	}
	for i, entity := range l.TrustedEntities {
		if err := entity.validate(); err != nil {
			return fmt.Errorf("trustedEntities[%d]: %w", i, err)
		}
	}
	for i, ptr := range l.PointersToOtherLoTEs {
		if err := ptr.validate(); err != nil {
			return fmt.Errorf("pointersToOtherLoTEs[%d]: %w", i, err)
		}
	}
	return nil
}

func (si *SchemeInformation) validate() error {
	if len(si.SchemeOperator) == 0 {
		return fmt.Errorf("schemeOperator is required")
	}
	if si.SchemeType == "" {
		return fmt.Errorf("schemeType is required")
	}
	if si.IssueDate.IsZero() {
		return fmt.Errorf("issueDate is required")
	}
	return nil
}

func (te *TrustedEntity) validate() error {
	if te.EntityID == "" {
		return fmt.Errorf("entityId is required")
	}
	if te.EntityStatus == "" {
		return fmt.Errorf("entityStatus is required")
	}
	for i, di := range te.DigitalIdentities {
		if err := di.validate(); err != nil {
			return fmt.Errorf("digitalIdentities[%d]: %w", i, err)
		}
	}
	for i, svc := range te.Services {
		if err := svc.validate(); err != nil {
			return fmt.Errorf("services[%d]: %w", i, err)
		}
	}
	return nil
}

func (di *DigitalIdentity) validate() error {
	if di.Type == "" {
		return fmt.Errorf("type is required")
	}
	if !validDigitalIdentityTypes[di.Type] {
		return fmt.Errorf("unknown type %q", di.Type)
	}
	switch di.Type {
	case "jwk":
		if len(di.JWK) == 0 {
			return fmt.Errorf("jwk must not be empty when type is \"jwk\"")
		}
	case "x509":
		if di.X509Certificate == "" {
			return fmt.Errorf("x509Certificate is required when type is \"x509\"")
		}
	case "did":
		if di.DID == "" {
			return fmt.Errorf("did is required when type is \"did\"")
		}
		if !strings.HasPrefix(di.DID, "did:") {
			return fmt.Errorf("did must start with \"did:\"")
		}
	case "x509_subject_name":
		if di.X509SubjectName == "" {
			return fmt.Errorf("x509SubjectName is required when type is \"x509_subject_name\"")
		}
	}
	return nil
}

func (es *EntityService) validate() error {
	if es.ServiceType == "" {
		return fmt.Errorf("serviceType is required")
	}
	if es.ServiceStatus == "" {
		return fmt.Errorf("serviceStatus is required")
	}
	return nil
}

func (lp *LoTEPointer) validate() error {
	if lp.Location == "" {
		return fmt.Errorf("location is required")
	}
	return nil
}
