module github.com/sirosfoundation/g119612

go 1.26

require (
	github.com/PuerkitoBio/goquery v1.12.0
	github.com/ThalesGroup/crypto11 v1.6.0
	github.com/beevik/etree v1.6.0
	github.com/go-jose/go-jose/v4 v4.1.4
	github.com/h2non/gock v1.2.0
	github.com/moov-io/signedxml v1.2.3
	github.com/russellhaering/goxmldsig v1.5.0
	github.com/sirosfoundation/go-cryptoutil v0.5.0
	github.com/sirosfoundation/go-cryptoutil/brainpool v0.2.0
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gematik/zero-lab/go/brainpool v0.0.0-20260309133150-5b2b80ad6517 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/miekg/pkcs11 v1.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/thales-e-security/pool v0.0.2 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

replace github.com/moov-io/signedxml v1.2.3 => github.com/sirosfoundation/signedxml v1.4.0-siros1

replace github.com/russellhaering/goxmldsig v1.5.0 => github.com/sirosfoundation/goxmldsig v1.6.0-siros3
