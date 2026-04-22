MODULE = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "1.0.0")
PACKAGES := $(shell go list ./... | grep -v /vendor/)
LDFLAGS := -ldflags "-X main.Version=${VERSION}"
GOBIN ?= $$(go env GOPATH)/bin

.PHONY: install-go-test-coverage
install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

.PHONY: check-coverage ## check test coverage and generate report
check-coverage: install-go-test-coverage ## generate coverage report
	go test -tags softhsm ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

.PHONY: default
default: build

# generate help info from comments: thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test:
	go test -tags softhsm -v ./...

.PHONY: build
build: tsl-tool ## build tsl-tool binary

.PHONY: tsl-tool
tsl-tool: ## build the tsl-tool binary
	go build ${LDFLAGS} -o tsl-tool -a cmd/tsl-tool/main.go

.PHONY: clean
clean: ## remove temporary files
	go clean
	rm -f *.out *.log tsl-tool

.PHONY: realclean
realclean: ## remove generated files - requires "make gen"
	rm -f pkg/etsi119612/*.xsd.go
	rm -f pkg/etsi119602/xmltypes/*.xsd.go

# horrid
.PHONY: gen
gen: gen-tsl gen-lote ## generate code from xsd

.PHONY: gen-tsl
gen-tsl: ## generate TS 119 612 (TSL) types from xsd
	xgen -i xsd2024 -o pkg/etsi119612 -l Go -p etsi119612
	sed -i 's/xml:lang/lang/g' pkg/etsi119612/*.xsd.go
	sed -i 's/tsl://g' pkg/etsi119612/*.xsd.go
	sed -i 's/*NonEmptyNormalizedString/*NonEmptyNormalizedString `xml:",chardata"`/g' pkg/etsi119612/*.xsd.go
	sed -i 's/*NonEmptyString/*NonEmptyString `xml:",chardata"`/g' pkg/etsi119612/*.xsd.go

.PHONY: gen-lote
gen-lote: ## generate TS 119 602 (LoTE) XML types from xsd
	xgen -i xsd_lote -o pkg/etsi119602/xmltypes -l Go -p xmltypes
	sed -i 's/xml:lang/lang/g' pkg/etsi119602/xmltypes/*.xsd.go
	sed -i 's/lote://g' pkg/etsi119602/xmltypes/*.xsd.go
	sed -i 's/tie://g' pkg/etsi119602/xmltypes/*.xsd.go
	sed -i 's/*NonEmptyNormalizedString/*NonEmptyNormalizedString `xml:",chardata"`/g' pkg/etsi119602/xmltypes/*.xsd.go
	sed -i 's/*NonEmptyString/*NonEmptyString `xml:",chardata"`/g' pkg/etsi119602/xmltypes/*.xsd.go
	sed -i '/^type AnyType struct {/{n;s/^}$$/\tContent string `xml:",chardata"`\n}/}' pkg/etsi119602/xmltypes/1960201_xsd_schema.xsd.go

gosec:
	$(info Run gosec)
	# G107 is excluded because where http.Get(url) is used the url can't be a constant.
	gosec -exclude=G107 -color -nosec -tests ./...

staticcheck:
	$(info Run staticcheck)
	staticcheck ./...


vscode:
	$(info Install APT packages)
	sudo apt-get update && sudo apt-get install -y \
		protobuf-compiler \
		netcat-openbsd
	$(info Install go packages)
	go install golang.org/x/tools/cmd/deadcode@latest && \
	go install github.com/securego/gosec/v2/cmd/gosec@latest && \
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/xuri/xgen/cmd/xgen@latest
