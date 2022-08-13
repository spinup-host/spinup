GOBIN ?= $(shell go env GOPATH)/bin
BINARY_NAME ?= spinup
VERSION ?= dev-unknown
DOCKER_NETWORK ?= "spinup_services"

SPINUP_BUILD_TAGS = -ldflags " \
			-X 'github.com/spinup-host/spinup/build.Version=$(VERSION)' \
			-X 'github.com/spinup-host/spinup/build.FullCommit=$(shell git rev-parse HEAD)' \
			-X 'github.com/spinup-host/spinup/build.Branch=$(shell git symbolic-ref --short HEAD)' \
			"

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all test build

all: help

build: ## Build your project and put the output binary in out/bin/
	go build $(SPINUP_BUILD_TAGS) -o bin/$(BINARY_NAME) .

clean: ## Remove build related file 
	rm -fr ./bin

test: 	## Run the tests of the project
	go test -v ./... $(OUTPUT_OPTIONS)

test-coverage: ## Run the tests of the project and export the coverage
	go test -cover -covermode=count -coverprofile=profile.cov ./...
	go tool cover -func profile.cov

format:
	gci write --section Standard --section Default --section "Prefix(github.com/spinup-host/spinup)" .

install-deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.2
	go install github.com/vektra/mockery/v2@v2.14.0
	go install github.com/daixiang0/gci@v0.4.2
	@echo 'Dev dependencies have been installed. Run "export PATH=$$PATH/$$(go env GOPATH)/bin" to use installed binaries.'

run-api:
	go run main.go start --api-only

checks:  ## Run all available checks and linters
	# golangci-lint run --enable-all # disable golangci-lint for now as it can get annoying

stop-services: ## Removes all running containers in the Spinup network
	docker stop $(shell docker container ls --filter="network=${DOCKER_NETWORK}" -q --all)

remove-services: ## Removes all running containers in the Spinup network
	docker rm $(shell docker container ls --filter="network=${DOCKER_NETWORK}" -q --all)


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