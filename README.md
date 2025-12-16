
[![Go Reference](https://pkg.go.dev/badge/github.com/sirosfoundation/g119612.svg)](https://pkg.go.dev/github.com/sirosfoundation/g119612)
[![Go Report Card](https://goreportcard.com/badge/github.com/sirosfoundation/g119612)](https://goreportcard.com/report/github.com/sirosfoundation/g119612)
![coverage](https://raw.githubusercontent.com/sirosfoundation/g119612/badges/.badges/main/coverage.svg)
[![License](https://img.shields.io/badge/License-BSD_2--Clause-orange.svg)](https://opensource.org/licenses/BSD-2-Clause)

# golang ETSI trust status lists (aka ETSI 119 612 v2)

This is a golang library implementing ETSI trust status lists. The library is meant to be used primarily to create a certificate pool for validating X509 certificates. The library was created to cater to the evolving EUDI wallet ecosystem but other uses are possible. Feel free to drop a PR or an issue if you see something you would like to change.

The library should be fully reentrant. There is no caching of URLs or other artefacts so make sure you fetch your TSLs from a CDN or similar and ensure availability.

## Basic Usage

The example below assumes you have imported the crypto/x509 and etsi119612 module (the latter from this package).

First step: fetch and create a TSL object
```go
    import (
        "github.com/sirosfoundation/g119612/pkg/etsi119612"
    )

    tsl, err := etsi119612.FetchTSL("https://example.com/some-tsl.xml")

    if err != nil {
	// do some error handling
    }
```

Next step: build a cert-pool from the trust status list with default validation policy
```go
    pool := tsl.ToCertPool(etsi119612.PolicyAll)
```

Finally: validate some cert
```go
    _, err = cert.Verify(x509.VerifyOptions{Roots: pool})
    if err != nil {
        //cert is INVALID
    }
```

## Trust List in the EUDI Infrastructure - General Overview:

Document for the reference:
https://github.com/EWC-consortium/eudi-wallet-rfcs/blob/main/ewc-rfc012-trust-mechanism.md#433-relying-parties

```mermaid
flowchart TD
    Issuer["Issuer"] -- Issues Credential --> Credential["Verifiable Credential"]
    Credential -- Stored in --> Wallet["Wallet Unit"]
    Wallet -- Presents Credential --> Verifier["Relying Party / Verifier"]
    Credential -- Includes --> Key["Public Key / Certificate"]
    Key -- Anchored in --> TL["Trusted List (EWC TL)"]
    Verifier -- Verifies Issuer & Credential --> TL
    Wallet -- Verifies Issuer & Credential --> TL
    Wallet -- Verifies Verifier --> TL
    Verifier -- Verifies Wallet Unit Attestation --> TL
    TL -. Must Register .-> Issuer & WalletProvider["Wallet Provider"]
    TL -. "Recommended to be Registered - Section 4.3.2.2" .-> Verifier

     Issuer:::actor
     Credential:::doc
     Wallet:::actor
     Verifier:::actor
     Key:::key
     TL:::trustlist
    classDef trustlist fill:#fdf6b2,stroke:#d97706,color:#92400e
    classDef actor fill:#f0f9ff,stroke:#0284c7,color:#0c4a6e
    classDef doc fill:#f3f4f6,stroke:#6b7280,color:#374151
    classDef key fill:#ecfccb,stroke:#65a30d,color:#365314
    linkStyle 1 stroke:#000000
```

## Contributing

If you want to "make gen" to re-generate the golang from the etsi XSD then you must install https://github.com/xuri/xgen first. Note that the generated code is post-processed (sed) to fix a couple of "features" in xgen that I am too lazy to pursue as bugs in xgen at this point. This stuff may change so run "make gen" at your own peril. The generated code that is known to work is commited into the repo for this reason - ymmw.

