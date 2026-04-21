// Package etsi119602 — this file implements XML marshaling and unmarshaling
// for LoTE and LoTL documents per ETSI TS 119 602.
//
// The XML representation uses the XSD-generated types from the xmltypes sub-package,
// produced from the official ETSI TS 119 602 XSD schema via xgen.
//
// The XML uses the namespace: http://uri.etsi.org/019602/v1#
package etsi119602

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119602/xmltypes"
)

const (
	// XMLNamespace is the XML namespace for ETSI TS 119 602.
	XMLNamespace = "http://uri.etsi.org/019602/v1#"
	// LOTETag is the LoTE tag attribute value.
	LOTETag = "http://uri.etsi.org/019602/LOTETag"
)

// --- Marshaling: LoTE → XML ---

// EncodeXML serializes the LoTE to XML bytes using XSD-generated types.
func (l *ListOfTrustedEntities) EncodeXML() ([]byte, error) {
	x := toXMLLoTE(l)
	return xml.MarshalIndent(x, "", "  ")
}

// EncodeXMLToFile writes the LoTE as XML to a file.
func (l *ListOfTrustedEntities) EncodeXMLToFile(path string) error {
	data, err := l.EncodeXML()
	if err != nil {
		return err
	}
	header := []byte(xml.Header)
	return os.WriteFile(path, append(header, data...), 0640)
}

// ParseLoTEXML parses a LoTE from XML bytes.
func ParseLoTEXML(data []byte) (*ListOfTrustedEntities, error) {
	var x xmltypes.ListOfTrustedEntitiesType
	if err := xml.Unmarshal(data, &x); err != nil {
		return nil, fmt.Errorf("failed to parse LoTE XML: %w", err)
	}
	return fromXMLLoTE(&x)
}

// ParseLoTEXMLFromFile loads and parses a LoTE from an XML file.
func ParseLoTEXMLFromFile(path string) (*ListOfTrustedEntities, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read LoTE XML file %s: %w", path, err)
	}
	return ParseLoTEXML(data)
}

// --- Marshaling: LoTL → XML ---

// EncodeXML serializes the LoTL to XML bytes.
func (l *ListOfTrustedLists) EncodeXML() ([]byte, error) {
	lote := l.ToLoTE()
	return lote.EncodeXML()
}

// EncodeXMLToFile writes the LoTL as XML to a file.
func (l *ListOfTrustedLists) EncodeXMLToFile(path string) error {
	data, err := l.EncodeXML()
	if err != nil {
		return err
	}
	header := []byte(xml.Header)
	return os.WriteFile(path, append(header, data...), 0640)
}

// ParseLoTLXML parses a LoTL from XML bytes.
func ParseLoTLXML(data []byte) (*ListOfTrustedLists, error) {
	lote, err := ParseLoTEXML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LoTL XML: %w", err)
	}
	return LoTLFromLoTE(lote), nil
}

// ParseLoTLXMLFromFile loads and parses a LoTL from an XML file.
func ParseLoTLXMLFromFile(path string) (*ListOfTrustedLists, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read LoTL XML file %s: %w", path, err)
	}
	return ParseLoTLXML(data)
}

// --- Internal conversion: domain types ↔ XSD-generated types ---
//
// NOTE: The following XSD fields are present in the generated types but are NOT
// currently mapped to/from the JSON domain model. They are preserved by the XSD
// layer on parse but lost when converting to/from the domain types:
//   - SchemeOperatorAddress (postal/electronic addresses)
//   - SchemeTypeCommunityRules
//   - HistoricalInformationPeriod
//   - TETradeName, TEAddress
//   - SchemeServiceDefinitionURI, TEServiceDefinitionURI
//   - ServiceHistory
//   - SchemeExtensions, TEInformationExtensions, ServiceInformationExtensions
//
// The entity-level properties (EntityType, EntityStatus, DigitalIdentities) from
// the JSON domain model are mapped to the first XSD TrustedEntityService. When
// parsing external XML, the first service's properties are promoted to entity-level
// fields and remaining services become entity.Services. This means external XML
// that has only one service per entity will have empty entity.Services after parsing.

func toXMLLoTE(l *ListOfTrustedEntities) *xmltypes.ListOfTrustedEntitiesType {
	schemeInfo := &xmltypes.LoTEListAndSchemeInformationType{
		LoTEVersionIdentifier:       1,
		LoTESequenceNumber:          l.SchemeInformation.SequenceNumber,
		LoteLoTEType:                l.SchemeInformation.SchemeType,
		LoteSchemeOperatorName:      nameSetToXMLNames(l.SchemeInformation.SchemeOperator),
		StatusDeterminationApproach: l.SchemeInformation.StatusDeterminationApproach,
		ListIssueDateTime:           l.SchemeInformation.IssueDate.UTC().Format(time.RFC3339),
	}

	if l.SchemeInformation.Territory != "" {
		territory := l.SchemeInformation.Territory
		schemeInfo.LoteSchemeTerritory = &territory
	}
	if len(l.SchemeInformation.SchemeName) > 0 {
		schemeInfo.LoteSchemeName = nameSetToXMLNames(l.SchemeInformation.SchemeName)
	}
	if len(l.SchemeInformation.SchemeInformationURI) > 0 {
		schemeInfo.LoteSchemeInformationURI = langURIsToXMLURIList(l.SchemeInformation.SchemeInformationURI)
	}
	if len(l.SchemeInformation.PolicyOrLegalNotice) > 0 {
		schemeInfo.LotePolicyOrLegalNotice = langStringsToXMLPolicy(l.SchemeInformation.PolicyOrLegalNotice)
	}
	if l.SchemeInformation.NextUpdate != nil {
		dt := l.SchemeInformation.NextUpdate.UTC().Format(time.RFC3339)
		schemeInfo.LoteNextUpdate = &xmltypes.NextUpdateType{DateTime: &dt}
	}
	if len(l.SchemeInformation.DistributionPoints) > 0 {
		schemeInfo.LoteDistributionPoints = &xmltypes.NonEmptyURIListType{
			URI: l.SchemeInformation.DistributionPoints,
		}
	}

	// Pointers
	if len(l.PointersToOtherLoTEs) > 0 {
		ptrs := &xmltypes.OtherLoTEPointersType{}
		for _, p := range l.PointersToOtherLoTEs {
			ptrs.LoteOtherLoTEPointer = append(ptrs.LoteOtherLoTEPointer, lotePointerToXML(p))
		}
		schemeInfo.LotePointersToOtherLoTE = ptrs
	}

	loteTag := l.LOTETag
	if loteTag == "" {
		loteTag = LOTETag
	}
	x := &xmltypes.ListOfTrustedEntitiesType{
		LOTETagAttr:                  loteTag,
		LoteListAndSchemeInformation: schemeInfo,
	}

	// Trusted entities
	if len(l.TrustedEntities) > 0 {
		tel := &xmltypes.TrustedEntitiesListType{}
		for _, e := range l.TrustedEntities {
			tel.LoteTrustedEntity = append(tel.LoteTrustedEntity, trustedEntityToXML(e))
		}
		x.LoteTrustedEntitiesList = tel
	}

	return x
}

func fromXMLLoTE(x *xmltypes.ListOfTrustedEntitiesType) (*ListOfTrustedEntities, error) {
	l := &ListOfTrustedEntities{
		Version: LoTEVersion,
		LOTETag: x.LOTETagAttr,
	}

	if si := x.LoteListAndSchemeInformation; si != nil {
		l.SchemeInformation.SchemeType = si.LoteLoTEType
		l.SchemeInformation.StatusDeterminationApproach = si.StatusDeterminationApproach
		l.SchemeInformation.SequenceNumber = si.LoTESequenceNumber

		if si.LoteSchemeTerritory != nil {
			l.SchemeInformation.Territory = *si.LoteSchemeTerritory
		}
		if si.LoteSchemeOperatorName != nil {
			l.SchemeInformation.SchemeOperator = xmlNamesToNameSet(si.LoteSchemeOperatorName)
		}
		if si.LoteSchemeName != nil {
			l.SchemeInformation.SchemeName = xmlNamesToNameSet(si.LoteSchemeName)
		}
		if si.LoteSchemeInformationURI != nil {
			l.SchemeInformation.SchemeInformationURI = xmlURIListToLangURIs(si.LoteSchemeInformationURI)
		}
		if si.LotePolicyOrLegalNotice != nil {
			l.SchemeInformation.PolicyOrLegalNotice = xmlPolicyToLangStrings(si.LotePolicyOrLegalNotice)
		}
		if si.ListIssueDateTime != "" {
			if t, err := time.Parse(time.RFC3339, si.ListIssueDateTime); err == nil {
				l.SchemeInformation.IssueDate = t
			}
		}
		if si.LoteNextUpdate != nil && si.LoteNextUpdate.DateTime != nil {
			if t, err := time.Parse(time.RFC3339, *si.LoteNextUpdate.DateTime); err == nil {
				l.SchemeInformation.NextUpdate = &t
			}
		}
		if si.LoteDistributionPoints != nil {
			l.SchemeInformation.DistributionPoints = si.LoteDistributionPoints.URI
		}

		// Pointers
		if si.LotePointersToOtherLoTE != nil {
			for _, xp := range si.LotePointersToOtherLoTE.LoteOtherLoTEPointer {
				l.PointersToOtherLoTEs = append(l.PointersToOtherLoTEs, xmlToLoTEPointer(xp))
			}
		}
	}

	// Trusted entities
	if x.LoteTrustedEntitiesList != nil {
		for _, te := range x.LoteTrustedEntitiesList.LoteTrustedEntity {
			l.TrustedEntities = append(l.TrustedEntities, xmlToTrustedEntity(te))
		}
	}

	return l, nil
}

// --- Name/URI helpers ---

func nameSetToXMLNames(ns NameSet) *xmltypes.InternationalNamesType {
	if len(ns) == 0 {
		return nil
	}
	x := &xmltypes.InternationalNamesType{}
	for _, n := range ns {
		lang := xmltypes.Lang(n.Language)
		value := xmltypes.NonEmptyNormalizedString(n.Value)
		x.Name = append(x.Name, &xmltypes.MultiLangNormStringType{
			XmlLangAttr:              &lang,
			NonEmptyNormalizedString: &value,
		})
	}
	return x
}

func xmlNamesToNameSet(x *xmltypes.InternationalNamesType) NameSet {
	if x == nil {
		return nil
	}
	var ns NameSet
	for _, n := range x.Name {
		ls := LangString{}
		if n.XmlLangAttr != nil {
			ls.Language = string(*n.XmlLangAttr)
		}
		if n.NonEmptyNormalizedString != nil {
			ls.Value = string(*n.NonEmptyNormalizedString)
		}
		ns = append(ns, ls)
	}
	return ns
}

func langURIsToXMLURIList(uris []LangURI) *xmltypes.NonEmptyMultiLangURIListType {
	if len(uris) == 0 {
		return nil
	}
	x := &xmltypes.NonEmptyMultiLangURIListType{}
	for _, u := range uris {
		lang := xmltypes.Lang(u.Language)
		x.URI = append(x.URI, &xmltypes.NonEmptyMultiLangURIType{
			XmlLangAttr: &lang,
			Value:       u.URI,
		})
	}
	return x
}

func xmlURIListToLangURIs(x *xmltypes.NonEmptyMultiLangURIListType) []LangURI {
	if x == nil {
		return nil
	}
	var uris []LangURI
	for _, u := range x.URI {
		lu := LangURI{URI: u.Value}
		if u.XmlLangAttr != nil {
			lu.Language = string(*u.XmlLangAttr)
		}
		uris = append(uris, lu)
	}
	return uris
}

func langStringsToXMLPolicy(strs []LangString) *xmltypes.PolicyOrLegalnoticeType {
	if len(strs) == 0 {
		return nil
	}
	x := &xmltypes.PolicyOrLegalnoticeType{}
	for _, s := range strs {
		lang := xmltypes.Lang(s.Language)
		// Use LoTEPolicy (URI) if the value looks like a URI, otherwise LoTELegalNotice (text)
		if strings.HasPrefix(s.Value, "http://") || strings.HasPrefix(s.Value, "https://") {
			x.LoTEPolicy = append(x.LoTEPolicy, &xmltypes.NonEmptyMultiLangURIType{
				XmlLangAttr: &lang,
				Value:       s.Value,
			})
		} else {
			value := xmltypes.NonEmptyString(s.Value)
			x.LoTELegalNotice = append(x.LoTELegalNotice, &xmltypes.MultiLangStringType{
				XmlLangAttr:    &lang,
				NonEmptyString: &value,
			})
		}
	}
	return x
}

func xmlPolicyToLangStrings(x *xmltypes.PolicyOrLegalnoticeType) []LangString {
	if x == nil {
		return nil
	}
	var strs []LangString
	// Read LoTEPolicy URIs
	for _, p := range x.LoTEPolicy {
		ls := LangString{Value: p.Value}
		if p.XmlLangAttr != nil {
			ls.Language = string(*p.XmlLangAttr)
		}
		strs = append(strs, ls)
	}
	// Read LoTELegalNotice text
	for _, n := range x.LoTELegalNotice {
		ls := LangString{}
		if n.XmlLangAttr != nil {
			ls.Language = string(*n.XmlLangAttr)
		}
		if n.NonEmptyString != nil {
			ls.Value = string(*n.NonEmptyString)
		}
		strs = append(strs, ls)
	}
	return strs
}

// --- Trusted entity conversion ---
//
// The JSON domain model has entity-level EntityType, EntityStatus, and DigitalIdentities.
// The XSD schema structures entities as TrustedEntityInformation + TrustedEntityServices.
// Entity-level properties are mapped to a dedicated first service; additional services
// from the domain model are appended after it.

func trustedEntityToXML(e TrustedEntity) *xmltypes.TEType {
	te := &xmltypes.TEType{
		LoteTrustedEntityInformation: &xmltypes.TrustedEntityInformationType{
			TEName: nameSetToXMLNames(e.EntityName),
		},
	}

	// EntityID and InformationURIs → TEInformationURI
	var infoURIs []LangURI
	if e.EntityID != "" {
		infoURIs = append(infoURIs, LangURI{Language: "en", URI: e.EntityID})
	}
	infoURIs = append(infoURIs, e.InformationURIs...)
	if len(infoURIs) > 0 {
		te.LoteTrustedEntityInformation.TEInformationURI = langURIsToXMLURIList(infoURIs)
	}

	// Build services list
	svcs := &xmltypes.TrustedEntityServicesListType{}

	// First service: entity-level properties (EntityType, EntityStatus, DigitalIdentities)
	entitySvc := &xmltypes.TrustedEntityServiceType{
		LoteServiceInformation: &xmltypes.TEServiceInformationType{
			ServiceName: nameSetToXMLNames(e.EntityName),
		},
	}
	if e.EntityType != "" {
		entitySvc.LoteServiceInformation.LoteServiceTypeIdentifier = &e.EntityType
	}
	if e.EntityStatus != "" {
		entitySvc.LoteServiceInformation.LoteServiceStatus = &e.EntityStatus
	}
	if e.StatusStartingTime != nil {
		st := e.StatusStartingTime.UTC().Format(time.RFC3339)
		entitySvc.LoteServiceInformation.StatusStartingTime = &st
	}
	if len(e.DigitalIdentities) > 0 {
		entitySvc.LoteServiceInformation.LoteServiceDigitalIdentity = digitalIdentitiesToXML(e.DigitalIdentities)
	}
	svcs.LoteTrustedEntityService = append(svcs.LoteTrustedEntityService, entitySvc)

	// Additional services from the domain model
	for _, s := range e.Services {
		svcs.LoteTrustedEntityService = append(svcs.LoteTrustedEntityService, entityServiceToXML(s))
	}

	te.LoteTrustedEntityServices = svcs
	return te
}

func xmlToTrustedEntity(te *xmltypes.TEType) TrustedEntity {
	e := TrustedEntity{}

	if te.LoteTrustedEntityInformation != nil {
		e.EntityName = xmlNamesToNameSet(te.LoteTrustedEntityInformation.TEName)

		// TEInformationURI → EntityID (first) + InformationURIs (rest)
		uris := xmlURIListToLangURIs(te.LoteTrustedEntityInformation.TEInformationURI)
		if len(uris) > 0 {
			e.EntityID = uris[0].URI
			if len(uris) > 1 {
				e.InformationURIs = uris[1:]
			}
		}
	}

	// Services: first service → entity-level properties, rest → entity.Services
	if te.LoteTrustedEntityServices != nil {
		services := te.LoteTrustedEntityServices.LoteTrustedEntityService
		if len(services) > 0 {
			first := services[0]
			if first.LoteServiceInformation != nil {
				si := first.LoteServiceInformation
				if si.LoteServiceTypeIdentifier != nil {
					e.EntityType = *si.LoteServiceTypeIdentifier
				}
				if si.LoteServiceStatus != nil {
					e.EntityStatus = *si.LoteServiceStatus
				}
				if si.StatusStartingTime != nil {
					if t, err := time.Parse(time.RFC3339, *si.StatusStartingTime); err == nil {
						e.StatusStartingTime = &t
					}
				}
				if si.LoteServiceDigitalIdentity != nil {
					e.DigitalIdentities = xmlToDigitalIdentities(si.LoteServiceDigitalIdentity)
				}
			}
			// Remaining services → entity.Services
			for _, xs := range services[1:] {
				e.Services = append(e.Services, xmlToEntityService(xs))
			}
		}
	}

	return e
}

// --- Service conversion ---

func entityServiceToXML(s EntityService) *xmltypes.TrustedEntityServiceType {
	svc := &xmltypes.TrustedEntityServiceType{
		LoteServiceInformation: &xmltypes.TEServiceInformationType{},
	}
	si := svc.LoteServiceInformation

	if s.ServiceType != "" {
		si.LoteServiceTypeIdentifier = &s.ServiceType
	}
	if len(s.ServiceName) > 0 {
		si.ServiceName = nameSetToXMLNames(s.ServiceName)
	}
	if s.ServiceStatus != "" {
		si.LoteServiceStatus = &s.ServiceStatus
	}
	if s.StatusStartingTime != nil {
		st := s.StatusStartingTime.UTC().Format(time.RFC3339)
		si.StatusStartingTime = &st
	}
	if len(s.DigitalIdentities) > 0 {
		si.LoteServiceDigitalIdentity = digitalIdentitiesToXML(s.DigitalIdentities)
	}
	if len(s.ServiceSupplyPoints) > 0 {
		pts := &xmltypes.ServiceSupplyPointsType{}
		for _, p := range s.ServiceSupplyPoints {
			pts.ServiceSupplyPoint = append(pts.ServiceSupplyPoint, &xmltypes.AttributedNonEmptyURIType{
				Value: p,
			})
		}
		si.LoteServiceSupplyPoints = pts
	}

	return svc
}

func xmlToEntityService(xs *xmltypes.TrustedEntityServiceType) EntityService {
	s := EntityService{}
	if xs.LoteServiceInformation == nil {
		return s
	}
	si := xs.LoteServiceInformation

	if si.LoteServiceTypeIdentifier != nil {
		s.ServiceType = *si.LoteServiceTypeIdentifier
	}
	if si.ServiceName != nil {
		s.ServiceName = xmlNamesToNameSet(si.ServiceName)
	}
	if si.LoteServiceStatus != nil {
		s.ServiceStatus = *si.LoteServiceStatus
	}
	if si.StatusStartingTime != nil {
		if t, err := time.Parse(time.RFC3339, *si.StatusStartingTime); err == nil {
			s.StatusStartingTime = &t
		}
	}
	if si.LoteServiceDigitalIdentity != nil {
		s.DigitalIdentities = xmlToDigitalIdentities(si.LoteServiceDigitalIdentity)
	}
	if si.LoteServiceSupplyPoints != nil {
		for _, p := range si.LoteServiceSupplyPoints.ServiceSupplyPoint {
			s.ServiceSupplyPoints = append(s.ServiceSupplyPoints, p.Value)
		}
	}

	return s
}

// --- Digital identity conversion ---
//
// XSD DigitalIdentityType is a choice of: X509Certificate, X509SubjectName,
// KeyValue, X509SKI, or OtherId. DID and JWK are mapped via OtherId.

func digitalIdentitiesToXML(ids []DigitalIdentity) *xmltypes.DigitalIdentityListType {
	x := &xmltypes.DigitalIdentityListType{}
	for _, id := range ids {
		x.DigitalId = append(x.DigitalId, digitalIdentityToXML(id))
	}
	return x
}

func digitalIdentityToXML(id DigitalIdentity) *xmltypes.DigitalIdentityType {
	xid := &xmltypes.DigitalIdentityType{}
	switch id.Type {
	case "x509":
		xid.X509Certificate = &id.X509Certificate
	case "x509_subject_name":
		xid.X509SubjectName = &id.X509SubjectName
	case "did":
		xid.OtherId = &xmltypes.AnyType{Content: id.DID}
	case "jwk":
		if len(id.JWK) > 0 {
			if jwkBytes, err := json.Marshal(id.JWK); err == nil {
				xid.OtherId = &xmltypes.AnyType{Content: string(jwkBytes)}
			}
		}
	}
	return xid
}

func xmlToDigitalIdentities(x *xmltypes.DigitalIdentityListType) []DigitalIdentity {
	if x == nil {
		return nil
	}
	var ids []DigitalIdentity
	for _, xid := range x.DigitalId {
		if id, ok := xmlToDigitalIdentity(xid); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func xmlToDigitalIdentity(xid *xmltypes.DigitalIdentityType) (DigitalIdentity, bool) {
	if xid == nil {
		return DigitalIdentity{}, false
	}
	if xid.X509Certificate != nil {
		return DigitalIdentity{
			Type:            "x509",
			X509Certificate: *xid.X509Certificate,
		}, true
	}
	if xid.X509SubjectName != nil {
		return DigitalIdentity{
			Type:            "x509_subject_name",
			X509SubjectName: *xid.X509SubjectName,
		}, true
	}
	if xid.OtherId != nil && xid.OtherId.Content != "" {
		content := xid.OtherId.Content
		if strings.HasPrefix(content, "did:") {
			return DigitalIdentity{Type: "did", DID: content}, true
		}
		if strings.HasPrefix(content, "{") {
			var jwk map[string]any
			if err := json.Unmarshal([]byte(content), &jwk); err == nil {
				return DigitalIdentity{Type: "jwk", JWK: jwk}, true
			}
		}
		return DigitalIdentity{Type: "other", DID: content}, true
	}
	return DigitalIdentity{}, false
}

// --- Pointer conversion ---
//
// The XSD OtherLoTEPointerType has LoTELocation, ServiceDigitalIdentities,
// and AdditionalInformation. SchemeTerritory and SchemeType from the JSON
// domain model are stored in AdditionalInformation.TextualInformation.

func lotePointerToXML(p LoTEPointer) *xmltypes.OtherLoTEPointerType {
	xp := &xmltypes.OtherLoTEPointerType{
		LoTELocation: p.Location,
	}

	if len(p.DigitalIdentities) > 0 {
		xp.LoteServiceDigitalIdentities = &xmltypes.ServiceDigitalIdentityListType{
			LoteServiceDigitalIdentity: []*xmltypes.DigitalIdentityListType{
				digitalIdentitiesToXML(p.DigitalIdentities),
			},
		}
	}

	// SchemeTerritory and SchemeType → AdditionalInformation.TextualInformation
	var textInfos []*xmltypes.MultiLangStringType
	if p.SchemeTerritory != "" {
		textInfos = append(textInfos, makeLangText("en", "SchemeTerritory:"+p.SchemeTerritory))
	}
	if p.SchemeType != "" {
		textInfos = append(textInfos, makeLangText("en", "SchemeType:"+p.SchemeType))
	}
	if len(textInfos) > 0 {
		xp.LoteAdditionalInformation = &xmltypes.AdditionalInformationType{
			TextualInformation: textInfos,
		}
	}

	return xp
}

func xmlToLoTEPointer(xp *xmltypes.OtherLoTEPointerType) LoTEPointer {
	p := LoTEPointer{
		Location: xp.LoTELocation,
	}

	if xp.LoteServiceDigitalIdentities != nil {
		for _, sdil := range xp.LoteServiceDigitalIdentities.LoteServiceDigitalIdentity {
			p.DigitalIdentities = append(p.DigitalIdentities, xmlToDigitalIdentities(sdil)...)
		}
	}

	// Extract SchemeTerritory and SchemeType from AdditionalInformation
	if xp.LoteAdditionalInformation != nil {
		for _, ti := range xp.LoteAdditionalInformation.TextualInformation {
			if ti.NonEmptyString == nil {
				continue
			}
			text := string(*ti.NonEmptyString)
			if after, ok := strings.CutPrefix(text, "SchemeTerritory:"); ok {
				p.SchemeTerritory = after
			}
			if after, ok := strings.CutPrefix(text, "SchemeType:"); ok {
				p.SchemeType = after
			}
		}
	}

	return p
}

func makeLangText(lang, value string) *xmltypes.MultiLangStringType {
	l := xmltypes.Lang(lang)
	v := xmltypes.NonEmptyString(value)
	return &xmltypes.MultiLangStringType{
		XmlLangAttr:    &l,
		NonEmptyString: &v,
	}
}
