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
	go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

.PHONY: default
default: build

# generate help info from comments: thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:  ## build the library
	CGO_ENABLED=0 go build ${LDFLAGS} -o etsi_ts -a cmd/etsi_ts/main.go

.PHONY: clean
clean: ## remove temporary files
	go clean
	rm -f *.out *.log

.PHONY: realclean
realclean: ## remove generated files - requires "make gen"
	rm -f pkg/etsi119612/*.xsd.go

# horrid
.PHONY: gen
gen: ## generate code from xsd
	xgen -i xsd2024 -o pkg/etsi119612 -l Go -p etsi119612
	sed -i 's/xml:lang/lang/g' pkg/etsi119612/*.xsd.go
	sed -i 's/tsl://g' pkg/etsi119612/*.xsd.go
	sed -i 's/*NonEmptyNormalizedString/*NonEmptyNormalizedString `xml:",chardata"`/g' pkg/etsi119612/*.xsd.go
	sed -i 's/*NonEmptyString/*NonEmptyString `xml:",chardata"`/g' pkg/etsi119612/*.xsd.go

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
