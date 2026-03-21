package etsi119602

import (
	"fmt"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// ServiceStatusHistory records a historical status change for an entity.
type ServiceStatusHistory struct {
	// ServiceType is the type identifier at the time of this status.
	ServiceType string `json:"serviceType,omitempty"`

	// ServiceName is the service name at the time of this status.
	ServiceName NameSet `json:"serviceName,omitempty"`

	// ServiceStatus is the status value.
	ServiceStatus string `json:"serviceStatus"`

	// StatusStartingTime is when this status became effective.
	StatusStartingTime *time.Time `json:"statusStartingTime,omitempty"`
}

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
				// Extract territory and additional info from AdditionalInformation
				if ptr.TslAdditionalInformation != nil {
					addlInfo := make(map[string]any)
					if len(ptr.TslAdditionalInformation.TextualInformation) > 0 {
						var texts []map[string]string
						for _, txt := range ptr.TslAdditionalInformation.TextualInformation {
							entry := map[string]string{}
							if txt.XmlLangAttr != nil {
								entry["language"] = string(*txt.XmlLangAttr)
							}
							if txt.NonEmptyString != nil {
								entry["value"] = string(*txt.NonEmptyString)
							}
							texts = append(texts, entry)
						}
						addlInfo["textualInformation"] = texts
					}
					if len(addlInfo) > 0 {
						lotePtr.AdditionalInformation = addlInfo
					}
				}
				// Extract signer identity from pointer
				if ptr.TslServiceDigitalIdentities != nil {
					for _, sdil := range ptr.TslServiceDigitalIdentities.TslServiceDigitalIdentity {
						if sdil == nil {
							continue
						}
						for _, did := range sdil.DigitalId {
							if did.X509Certificate != "" {
								lotePtr.DigitalIdentities = append(lotePtr.DigitalIdentities, DigitalIdentity{
									Type:            "x509",
									X509Certificate: did.X509Certificate,
								})
							}
						}
					}
				}
				lote.PointersToOtherLoTEs = append(lote.PointersToOtherLoTEs, lotePtr)
			}
		}
	}

	// Convert trust service providers to trusted entities
	if tsl.StatusList.TslTrustServiceProviderList != nil {
		tspIndex := 0
		for _, tsp := range tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider {
			if tsp.TslTSPServices == nil {
				continue
			}

			// Extract provider-level metadata for all entities from this TSP
			var providerName NameSet
			var providerURIs []LangURI
			var providerExtensions map[string]any

			if tsp.TslTSPInformation != nil {
				providerName = internationalNamesToNameSet(tsp.TslTSPInformation.TSPName)
				providerURIs = urisFromInternational(tsp.TslTSPInformation.TSPInformationURI)

				// Convert address info to extensions
				if tsp.TslTSPInformation.TSPAddress != nil {
					addrExt := convertAddress(tsp.TslTSPInformation.TSPAddress)
					if len(addrExt) > 0 {
						providerExtensions = map[string]any{
							"tsp_address": addrExt,
						}
					}
				}

				// Include trade name if present
				if tsp.TslTSPInformation.TSPTradeName != nil {
					tradeName := internationalNamesToNameSet(tsp.TslTSPInformation.TSPTradeName)
					if len(tradeName) > 0 {
						if providerExtensions == nil {
							providerExtensions = make(map[string]any)
						}
						providerExtensions["tsp_trade_name"] = tradeName
					}
				}
			}

			svcIndex := 0
			for _, svc := range tsp.TslTSPServices.TslTSPService {
				if svc.TslServiceInformation == nil {
					continue
				}
				si := svc.TslServiceInformation

				// Build unique entity ID: serviceType + tsp index + svc index
				entityID := fmt.Sprintf("%s#tsp%d-svc%d", si.TslServiceTypeIdentifier, tspIndex, svcIndex)

				entity := TrustedEntity{
					EntityID:     entityID,
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

				// Provider information URIs
				for _, uri := range providerURIs {
					entity.InformationURIs = append(entity.InformationURIs, uri)
				}

				// Build extensions from provider data and service history
				ext := make(map[string]any)

				// Include provider name in extensions
				if len(providerName) > 0 {
					ext["tsp_name"] = providerName
				}

				// Copy provider-level extensions
				for k, v := range providerExtensions {
					ext[k] = v
				}

				// Convert service history
				if svc.TslServiceHistory != nil {
					var history []ServiceStatusHistory
					for _, hist := range svc.TslServiceHistory.TslServiceHistoryInstance {
						entry := ServiceStatusHistory{
							ServiceType:   hist.TslServiceTypeIdentifier,
							ServiceName:   internationalNamesToNameSet(hist.ServiceName),
							ServiceStatus: hist.TslServiceStatus,
						}
						if hist.StatusStartingTime != "" {
							if t, err := time.Parse(time.RFC3339, hist.StatusStartingTime); err == nil {
								entry.StatusStartingTime = &t
							}
						}
						history = append(history, entry)
					}
					if len(history) > 0 {
						ext["service_history"] = history
					}
				}

				// Convert service supply points
				if si.TslServiceSupplyPoints != nil && si.TslServiceSupplyPoints.ServiceSupplyPoint != nil {
					entity.Services = append(entity.Services, EntityService{
						ServiceType:         si.TslServiceTypeIdentifier,
						ServiceName:         internationalNamesToNameSet(si.ServiceName),
						ServiceStatus:       si.TslServiceStatus,
						ServiceSupplyPoints: []string{si.TslServiceSupplyPoints.ServiceSupplyPoint.Value},
					})
				}

				if len(ext) > 0 {
					entity.Extensions = ext
				}

				lote.TrustedEntities = append(lote.TrustedEntities, entity)
				svcIndex++
			}
			tspIndex++
		}
	}

	return lote
}

// convertAddress converts TSL AddressType to a map for extensions.
func convertAddress(addr *etsi119612.AddressType) map[string]any {
	result := make(map[string]any)

	if addr.TslPostalAddresses != nil && len(addr.TslPostalAddresses.TslPostalAddress) > 0 {
		var addrs []map[string]string
		for _, pa := range addr.TslPostalAddresses.TslPostalAddress {
			entry := map[string]string{
				"streetAddress": pa.StreetAddress,
				"locality":      pa.Locality,
				"countryName":   pa.CountryName,
			}
			if pa.PostalCode != "" {
				entry["postalCode"] = pa.PostalCode
			}
			if pa.StateOrProvince != "" {
				entry["stateOrProvince"] = pa.StateOrProvince
			}
			if pa.XmlLangAttr != nil {
				entry["language"] = string(*pa.XmlLangAttr)
			}
			addrs = append(addrs, entry)
		}
		result["postal"] = addrs
	}

	if addr.TslElectronicAddress != nil && len(addr.TslElectronicAddress.URI) > 0 {
		var uris []LangURI
		for _, u := range addr.TslElectronicAddress.URI {
			lang := ""
			if u.XmlLangAttr != nil {
				lang = string(*u.XmlLangAttr)
			}
			uris = append(uris, LangURI{Language: lang, URI: u.Value})
		}
		result["electronic"] = uris
	}

	return result
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
