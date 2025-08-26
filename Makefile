.PHONY: build

build:
	go build -o ./build/orbgen .

install:
	go install -o ./build/orbgen .

#=============================================================================#
#                                 Tooling                                     #
#=============================================================================#
.PHONY: tool-all license format lint nancy

# This runs all the common tools like linting, etc.
tool-all: license format lint nancy

FILES := $(shell find . -name "*.go" -not -path "./simapp/*" -not -name "*.pb.go" -not -name "*.pb.gw.go" -not -name "*.pulsar.go")
license:
	@echo "Adding license to files..."
	@go-license --config .github/license.yaml $(FILES)
	@echo "Completed license addition!"

check-license:
	@echo "Checking files for license..."
	@go-license --config .github/license.yaml $(FILES) --verify
	@echo "Done!"

GOLANGCI_LINT_VERSION="v2.2.2"
GOLANGCI_LINT_IMAGE=golangci/golangci-lint:$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT_CMD=docker run --rm -v $(PWD):/app -w /app $(GOLANGCI_LINT_IMAGE) golangci-lint
format:
	@echo "Running formatters..."
	@$(GOLANGCI_LINT_CMD) fmt -c ./.golangci.yaml
	@echo "Completed formatting!"

lint:
	@echo "Running linter..."
	@$(GOLANGCI_LINT_CMD) run -c ./.golangci.yaml
	@$(MAKE) check-license
	@echo "Completed linting!"

NANCY_VERSION=v1.0
NANCY_IMAGE=sonatypecommunity/nancy:$(NANCY_VERSION)
NANCY_CMD=docker run --rm -i --volume "$(PWD)":/app --workdir /app $(NANCY_IMAGE)
nancy:
	@echo "Running Nancy vulnerability scanner..."
	@go list -json -deps ./... | $(NANCY_CMD) sleuth --exclude-vulnerability-file .nancy-ignore
	@echo "Completed Nancy vulnerability scan!"

