module github.com/sirosfoundation/g119612

go 1.23.0

toolchain go1.23.2

require (
	github.com/h2non/gock v1.2.0
	github.com/moov-io/signedxml v1.2.3
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/beevik/etree v1.5.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russellhaering/goxmldsig v1.5.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/moov-io/signedxml v1.2.3 => github.com/leifj/signedxml v1.2.3-leifj2
