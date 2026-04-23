package etsi119602

import (
	"fmt"
)

// Validate checks the LoTE document for structural correctness.
func (l *ListOfTrustedEntities) Validate() error {
	if err := l.ListAndSchemeInformation.validate(); err != nil {
		return fmt.Errorf("ListAndSchemeInformation: %w", err)
	}
	for i, entity := range l.TrustedEntitiesList {
		if err := entity.validate(); err != nil {
			return fmt.Errorf("TrustedEntitiesList[%d]: %w", i, err)
		}
	}
	for i, ptr := range l.ListAndSchemeInformation.PointersToOtherLoTE {
		if err := ptr.validate(); err != nil {
			return fmt.Errorf("ListAndSchemeInformation.PointersToOtherLoTE[%d]: %w", i, err)
		}
	}
	return nil
}

func (si *ListAndSchemeInformation) validate() error {
	if len(si.SchemeOperatorName) == 0 {
		return fmt.Errorf("SchemeOperatorName is required")
	}
	if si.ListIssueDateTime == "" {
		return fmt.Errorf("ListIssueDateTime is required")
	}
	if si.NextUpdate == "" {
		return fmt.Errorf("NextUpdate is required")
	}
	return nil
}

func (te *TrustedEntity) validate() error {
	if len(te.TrustedEntityInformation.TEName) == 0 {
		return fmt.Errorf("TrustedEntityInformation.TEName is required")
	}
	if te.TrustedEntityInformation.TEAddress == nil {
		return fmt.Errorf("TrustedEntityInformation.TEAddress is required")
	}
	if len(te.TrustedEntityInformation.TEInformationURI) == 0 {
		return fmt.Errorf("TrustedEntityInformation.TEInformationURI is required")
	}
	if len(te.TrustedEntityServices) == 0 {
		return fmt.Errorf("TrustedEntityServices is required")
	}
	for i, svc := range te.TrustedEntityServices {
		if err := svc.validate(); err != nil {
			return fmt.Errorf("TrustedEntityServices[%d]: %w", i, err)
		}
	}
	return nil
}

func (svc *TrustedEntityService) validate() error {
	if len(svc.ServiceInformation.ServiceName) == 0 {
		return fmt.Errorf("ServiceInformation.ServiceName is required")
	}
	return nil
}

func (ptr *OtherLoTEPointer) validate() error {
	if ptr.LoTELocation == "" {
		return fmt.Errorf("LoTELocation is required")
	}
	return nil
}
