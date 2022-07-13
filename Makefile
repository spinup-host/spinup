GOBIN ?= $(shell go env GOPATH)/bin
BINARY_NAME ?= spinup
VERSION ?= dev-unknown
SERVICE_PORT ?= 4434

DOCKER_REGISTRY?= #if set it should finished by /
EXPORT_RESULT?=false # for CI please set EXPORT_RESULT to true

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all test build

all: help

build: ## Build your project and put the output binary in out/bin/
	go build -o bin/$(BINARY_NAME) .

clean: ## Remove build related file 
	rm -fr ./bin

test: 	## Run the tests of the project
	go test -v ./... $(OUTPUT_OPTIONS)

test-coverage: ## Run the tests of the project and export the coverage
	go test -cover -covermode=count -coverprofile=profile.cov ./...
	go tool cover -func profile.cov

install-deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.2
	go install github.com/vektra/mockery/v2@v2.14.0
	@echo 'Dev dependencies have been installed. Run "export PATH=$$PATH/$$(go env GOPATH)/bin" to use installed binaries.'

run-api:
	go run main.go start --api-only

checks:  ## Run all available checks and linters
	# golangci-lint run --enable-all # disable golangci-lint for now as it can get annoying


help: ## Show make commands help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)