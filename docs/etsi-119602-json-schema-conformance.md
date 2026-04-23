# ETSI TS 119 602-1 JSON Schema Conformance Analysis

**Date:** 2026-04-23

**Reference Schema:** `1960201_json_schema.json` from
[forge.etsi.org/rep/esi/x19_60201_lists_of_trusted_entities](https://forge.etsi.org/rep/esi/x19_60201_lists_of_trusted_entities) (main branch)

This document compares two implementations against the official ETSI TS 119 602-1
JSON schema:

1. **g119612** (`sirosfoundation/g119612`, package `pkg/etsi119602`)
2. **wp4-trust-group LoTL tools** (`webuild-consortium/wp4-trust-group`, `tools/lotl/json_generator.py`)

---

## 1. Root Document Structure

**ETSI Schema:**
```json
{
  "type": "object",
  "required": ["LoTE"],
  "additionalProperties": false,
  "properties": {
    "LoTE": { "$ref": "#/definitions/LoTE" }
  }
}
```

| | g119612 | wp4-trust-group |
|---|---|---|
| Root wrapper `{"LoTE": {...}}` | Yes (`LoTEDocument` struct) | **No** — returns flat `{"loteTag": ..., "schemeInformation": ...}` |
| `additionalProperties: false` | Conforms (no extra top-level keys) | **Violates** — `loteTag` is not a valid property |

---

## 2. LoTE Object

**ETSI Schema:**
```json
"LoTE": {
  "properties": {
    "ListAndSchemeInformation": { ... },
    "TrustedEntitiesList": { ... }
  },
  "required": ["ListAndSchemeInformation"],
  "additionalProperties": false
}
```

| | g119612 | wp4-trust-group |
|---|---|---|
| `ListAndSchemeInformation` | Yes (PascalCase) | **No** — uses `schemeInformation` (camelCase, different name) |
| `TrustedEntitiesList` | Yes (omitempty) | Not produced (LoTL-only tool) |
| `additionalProperties: false` | Conforms | **Violates** — includes `loteTag` inside the object |

---

## 3. ListAndSchemeInformation Fields

**ETSI Schema required fields:** `LoTEVersionIdentifier`, `LoTESequenceNumber`,
`SchemeOperatorName`, `ListIssueDateTime`, `NextUpdate`

| ETSI Schema Field | g119612 JSON tag | wp4-trust-group key | Conforms (g119612) | Conforms (wp4) |
|---|---|---|---|---|
| `LoTEVersionIdentifier` | `"LoTEVersionIdentifier"` | `"loteVersionIdentifier"` | Yes | **No** (wrong case) |
| `LoTESequenceNumber` | `"LoTESequenceNumber"` | `"loteSequenceNumber"` | Yes | **No** (wrong case) |
| `LoTEType` | `"LoTEType"` | `"loteType"` | Yes | **No** (wrong case) |
| `SchemeOperatorName` | `"SchemeOperatorName"` | `"schemeOperatorName"` | Yes | **No** (wrong case) |
| `SchemeOperatorAddress` | `"SchemeOperatorAddress"` | `"schemeOperatorAddress"` | Yes | **No** (wrong case + wrong child field names, see §4) |
| `SchemeName` | `"SchemeName"` | `"schemeName"` | Yes | **No** (wrong case) |
| `SchemeInformationURI` | `"SchemeInformationURI"` | `"schemeInformationURI"` | Yes | **No** (wrong case) |
| `StatusDeterminationApproach` | `"StatusDeterminationApproach"` | `"statusDeterminationApproach"` | Yes | **No** (wrong case) |
| `SchemeTypeCommunityRules` | `"SchemeTypeCommunityRules"` | `"schemeTypeCommunityRules"` | Yes | **No** (wrong case) |
| `SchemeTerritory` | `"SchemeTerritory"` | `"schemeTerritory"` | Yes | **No** (wrong case) |
| `PolicyOrLegalNotice` | `"PolicyOrLegalNotice"` | Not produced | Yes | N/A |
| `HistoricalInformationPeriod` | `"HistoricalInformationPeriod"` | Not produced | Yes | N/A |
| `PointersToOtherLoTE` | `"PointersToOtherLoTE"` | **Not produced** — uses `"distributionPoints"` instead | Yes | **No** (wrong field name + wrong structure, see §5) |
| `ListIssueDateTime` | `"ListIssueDateTime"` | `"listIssueDateTime"` | Yes | **No** (wrong case) |
| `NextUpdate` | `"NextUpdate"` | `"nextUpdate"` | Yes | **No** (wrong case) |
| `DistributionPoints` | `"DistributionPoints"` | `"distributionPoints"` (but wrong structure) | Yes | **No** (wrong case + wrong value type, see §5) |
| `SchemeExtensions` | `"SchemeExtensions"` | Not produced | Yes | N/A |

---

## 4. SchemeOperatorAddress

**ETSI Schema:**
```json
"SchemeOperatorAddress": {
  "properties": {
    "SchemeOperatorPostalAddress": { "$ref": "#/definitions/PostalAddresses" },
    "SchemeOperatorElectronicAddress": { "$ref": "#/definitions/ElectronicAddress" }
  },
  "required": ["SchemeOperatorPostalAddress", "SchemeOperatorElectronicAddress"],
  "additionalProperties": false
}
```

| | g119612 | wp4-trust-group |
|---|---|---|
| `SchemeOperatorPostalAddress` | Yes (PascalCase) | **No** — uses `"postalAddresses"` |
| `SchemeOperatorElectronicAddress` | Yes (PascalCase) | **No** — uses `"electronicAddress"` |
| Electronic address items | `NonEmptyMultiLangURI` (`{lang, uriValue}`) | **No** — uses `{"lang": "en", "uri": "..."}` (`uri` instead of `uriValue`) |

---

## 5. DistributionPoints vs PointersToOtherLoTE

**ETSI Schema — `DistributionPoints`:**
```json
"DistributionPoints": {
  "type": "array",
  "items": { "type": "string", "format": "uri" }
}
```

**ETSI Schema — `PointersToOtherLoTE`:**
```json
"PointersToOtherLoTE": {
  "type": "array",
  "items": { "$ref": "#/definitions/OtherLoTEPointer" }
}
```

The wp4 tool conflates these two concepts. It produces a `"distributionPoints"` array
where each element is a complex object:

```json
{
  "tlType": "pid-provider",
  "referencedListTypeUri": "http://uri.etsi.org/19602/LoTEType/EUPIDProvidersList",
  "participantId": "...",
  "tlUrl": "...",
  "tlUrlJson": "...",
  "tlUrlXml": "...",
  "metadata": { ... }
}
```

**ETSI violations:**
- `DistributionPoints` items must be plain URI strings, not objects.
- The TL pointer information should be in `PointersToOtherLoTE` using `OtherLoTEPointer` objects.
- `OtherLoTEPointer` requires `{LoTELocation, ServiceDigitalIdentities, LoTEQualifiers}`.
- Fields like `tlType`, `participantId`, `tlUrl`, `tlUrlJson`, `tlUrlXml` are not defined in the ETSI schema.

| | g119612 | wp4-trust-group |
|---|---|---|
| `DistributionPoints` | Array of URI strings | **No** — array of complex objects |
| `PointersToOtherLoTE` | Array of `OtherLoTEPointer` | **Not produced** |

---

## 6. multiLangString

**ETSI Schema:**
```json
"multiLangString": {
  "properties": { "lang": { "type": "string" }, "value": { "type": "string" } },
  "required": ["lang", "value"],
  "additionalProperties": false
}
```

| | g119612 | wp4-trust-group |
|---|---|---|
| Field names | `{"lang": "en", "value": "..."}` | `{"lang": "en", "value": "..."}` |
| Conforms | Yes | Yes |

---

## 7. NonEmptyMultiLangURI

**ETSI Schema:**
```json
"NonEmptyMultiLangURI": {
  "properties": { "lang": { "type": "string" }, "uriValue": { "type": "string", "format": "uri" } },
  "required": ["lang", "uriValue"],
  "additionalProperties": false
}
```

| | g119612 | wp4-trust-group |
|---|---|---|
| Field names | `{"lang": "en", "uriValue": "..."}` | `{"lang": "en", "uri": "..."}` |
| Conforms | Yes | **No** — uses `"uri"` instead of `"uriValue"` |

---

## 8. LOTETag / loteTag

The JSON schema does not define a `LOTETag` or `loteTag` property. In the XML schema,
`LOTETag` is an attribute on the root `ListOfTrustedEntities` element. The JSON schema's
root object only allows `{"LoTE": {...}}` with `additionalProperties: false`.

| | g119612 | wp4-trust-group |
|---|---|---|
| `LOTETag` in JSON | Excluded from JSON via `json:"-"` tag | **Included** as `"loteTag"` at top level |
| Conforms | Yes | **No** — violates `additionalProperties: false` |

---

## 9. Summary

### g119612: Fully Conformant

All JSON field names match the official schema exactly. The `LoTE` root wrapper,
`ListAndSchemeInformation` structure, all type definitions (`multiLangString`,
`NonEmptyMultiLangURI`, `pkiOb`, `PostalAddress`, `ServiceDigitalIdentity`,
`OtherLoTEPointer`, `LoTEQualifier`, etc.) produce JSON that validates against
`1960201_json_schema.json`.

### wp4-trust-group `json_generator.py`: Non-Conformant

The tool uses an ad-hoc JSON structure that differs from the ETSI schema in
every field name (camelCase vs PascalCase), omits the required `{"LoTE": {...}}`
root wrapper, uses wrong sub-field names (`uri` vs `uriValue`, `postalAddresses` vs
`SchemeOperatorPostalAddress`), and replaces `PointersToOtherLoTE` with a custom
`distributionPoints` structure containing implementation-specific fields.

The wp4 XML generator (`xml_generator.py`) produces TS 119 612
`TrustServiceStatusList` XML, which is a different schema (the legacy trusted list
format) rather than TS 119 602-1 `ListOfTrustedEntities`.
