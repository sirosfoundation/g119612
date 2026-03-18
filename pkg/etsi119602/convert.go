package etsi119602

import (
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// FromTSL converts a TS 119 612 TrustStatusListType to a TS 119 602 ListOfTrustedEntities.
// This allows existing XML TSLs to be represented in the LoTE JSON format.
func FromTSL(tsl *etsi119612.TSL) *ListOfTrustedEntities {
	lote := &ListOfTrustedEntities{
		Version: LoTEVersion,
	}

	if tsl.StatusList.TslSchemeInformation != nil {
		si := tsl.StatusList.TslSchemeInformation
		lote.SchemeInformation = SchemeInformation{
			Territory:                   si.TslSchemeTerritory,
			SchemeOperator:              internationalNamesToNameSet(si.TslSchemeOperatorName),
			SchemeName:                  internationalNamesToNameSet(si.TslSchemeName),
			SchemeType:                  si.TslTSLType,
			SequenceNumber:              si.TSLSequenceNumber,
			StatusDeterminationApproach: si.StatusDeterminationApproach,
		}

		if si.ListIssueDateTime != "" {
			if t, err := time.Parse(time.RFC3339, si.ListIssueDateTime); err == nil {
				lote.SchemeInformation.IssueDate = t
			}
		}
		if si.TslNextUpdate != nil && si.TslNextUpdate.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, si.TslNextUpdate.DateTime); err == nil {
				lote.SchemeInformation.NextUpdate = &t
			}
		}
		if si.TslSchemeInformationURI != nil {
			for _, uri := range si.TslSchemeInformationURI.URI {
				lang := ""
				if uri.XmlLangAttr != nil {
					lang = string(*uri.XmlLangAttr)
				}
				lote.SchemeInformation.SchemeInformationURI = append(
					lote.SchemeInformation.SchemeInformationURI,
					LangURI{Language: lang, URI: uri.Value},
				)
			}
		}
		if si.TslDistributionPoints != nil {
			lote.SchemeInformation.DistributionPoints = si.TslDistributionPoints.URI
		}
		if si.TslPolicyOrLegalNotice != nil {
			for _, notice := range si.TslPolicyOrLegalNotice.TSLLegalNotice {
				lang := ""
				if notice.XmlLangAttr != nil {
					lang = string(*notice.XmlLangAttr)
				}
				val := ""
				if notice.NonEmptyString != nil {
					val = string(*notice.NonEmptyString)
				}
				lote.SchemeInformation.PolicyOrLegalNotice = append(
					lote.SchemeInformation.PolicyOrLegalNotice,
					LangString{Language: lang, Value: val},
				)
			}
		}

		// Convert pointers to other TSLs
		if si.TslPointersToOtherTSL != nil {
			for _, ptr := range si.TslPointersToOtherTSL.TslOtherTSLPointer {
				lotePtr := LoTEPointer{
					Location: ptr.TSLLocation,
				}
				// Extract territory from additional information if available
				lote.PointersToOtherLoTEs = append(lote.PointersToOtherLoTEs, lotePtr)
			}
		}
	}

	// Convert trust service providers to trusted entities
	if tsl.StatusList.TslTrustServiceProviderList != nil {
		for _, tsp := range tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider {
			if tsp.TslTSPServices == nil {
				continue
			}
			for _, svc := range tsp.TslTSPServices.TslTSPService {
				if svc.TslServiceInformation == nil {
					continue
				}
				si := svc.TslServiceInformation

				entity := TrustedEntity{
					EntityID:     si.TslServiceTypeIdentifier,
					EntityName:   internationalNamesToNameSet(si.ServiceName),
					EntityType:   si.TslServiceTypeIdentifier,
					EntityStatus: si.TslServiceStatus,
				}

				if si.StatusStartingTime != "" {
					if t, err := time.Parse(time.RFC3339, si.StatusStartingTime); err == nil {
						entity.StatusStartingTime = &t
					}
				}

				// Convert digital identities
				if si.TslServiceDigitalIdentity != nil {
					for _, did := range si.TslServiceDigitalIdentity.DigitalId {
						if did.X509Certificate != "" {
							entity.DigitalIdentities = append(entity.DigitalIdentities, DigitalIdentity{
								Type:            "x509",
								X509Certificate: did.X509Certificate,
							})
						}
						if did.X509SubjectName != "" {
							entity.DigitalIdentities = append(entity.DigitalIdentities, DigitalIdentity{
								Type:            "x509_subject_name",
								X509SubjectName: did.X509SubjectName,
							})
						}
					}
				}

				// Provider info as service-level data
				if tsp.TslTSPInformation != nil {
					for _, uri := range urisFromInternational(tsp.TslTSPInformation.TSPInformationURI) {
						entity.InformationURIs = append(entity.InformationURIs, uri)
					}
				}

				lote.TrustedEntities = append(lote.TrustedEntities, entity)
			}
		}
	}

	return lote
}

// internationalNamesToNameSet converts 119 612 InternationalNamesType to a NameSet.
func internationalNamesToNameSet(names *etsi119612.InternationalNamesType) NameSet {
	if names == nil {
		return nil
	}
	var ns NameSet
	for _, name := range names.Name {
		lang := ""
		if name.XmlLangAttr != nil {
			lang = string(*name.XmlLangAttr)
		}
		val := ""
		if name.NonEmptyNormalizedString != nil {
			val = string(*name.NonEmptyNormalizedString)
		}
		ns = append(ns, LangString{Language: lang, Value: val})
	}
	return ns
}

// urisFromInternational converts NonEmptyMultiLangURIListType to LangURI slice.
func urisFromInternational(uris *etsi119612.NonEmptyMultiLangURIListType) []LangURI {
	if uris == nil {
		return nil
	}
	var result []LangURI
	for _, uri := range uris.URI {
		lang := ""
		if uri.XmlLangAttr != nil {
			lang = string(*uri.XmlLangAttr)
		}
		result = append(result, LangURI{Language: lang, URI: uri.Value})
	}
	return result
}
