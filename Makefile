# k8s-yaml-diff Makefile

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet
BINARY_NAME=k8s-yaml-diff
BINARY_PATH=./cmd/k8s-yaml-diff

# Tool paths
BIN_DIR=$(CURDIR)/bin

# Build
.PHONY: build
build:
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(BINARY_PATH)

# Clean
.PHONY: clean
clean:
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
	rm -rf $(BIN_DIR)

# Dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) verify

.PHONY: tidy
tidy:
	$(GOMOD) tidy

# Tools
.PHONY: tools.init
tools.init:
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install golang.org/x/tools/cmd/goimports@latest
	GOBIN=$(BIN_DIR) go install honnef.co/go/tools/cmd/staticcheck@latest
	GOBIN=$(BIN_DIR) go install golang.org/x/lint/golint@latest
	GOBIN=$(BIN_DIR) go install github.com/gordonklaus/ineffassign@latest
	GOBIN=$(BIN_DIR) go install github.com/kisielk/errcheck@latest
	GOBIN=$(BIN_DIR) go install github.com/client9/misspell/cmd/misspell@latest
	GOBIN=$(BIN_DIR) go install github.com/securego/gosec/v2/cmd/gosec@latest
	GOBIN=$(BIN_DIR) go install github.com/mgechev/revive@latest
	GOBIN=$(BIN_DIR) go install gotest.tools/gotestsum@latest

# Linting
.PHONY: fmt
fmt:
	$(GOFMT) -w .
	$(GOCMD) fmt ./...
	@test -f $(BIN_DIR)/goimports || $(MAKE) tools.init
	$(BIN_DIR)/goimports -w .

.PHONY: fmt-check
fmt-check:
	@echo "Checking formatting..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Files need formatting. Run 'make fmt'" && $(GOFMT) -l . && exit 1)
	@echo "Checking imports formatting..."
	@test -f $(BIN_DIR)/goimports || $(MAKE) tools.init
	@test -z "$$($(BIN_DIR)/goimports -l .)" || (echo "Files need import formatting. Run 'make fmt'" && $(BIN_DIR)/goimports -l . && exit 1)

.PHONY: vet
vet:
	$(GOVET) ./...

.PHONY: lint
lint: fmt-check vet
	@echo "Running staticcheck..."
	@test -f $(BIN_DIR)/staticcheck || $(MAKE) tools.init
	$(BIN_DIR)/staticcheck ./...
	@echo "Running golint..."
	@test -f $(BIN_DIR)/golint || $(MAKE) tools.init
	$(BIN_DIR)/golint ./...
	@echo "Running ineffassign..."
	@test -f $(BIN_DIR)/ineffassign || $(MAKE) tools.init
	$(BIN_DIR)/ineffassign ./...
	@echo "Running errcheck..."
	@test -f $(BIN_DIR)/errcheck || $(MAKE) tools.init
	$(BIN_DIR)/errcheck ./...
	@echo "Running misspell..."
	@test -f $(BIN_DIR)/misspell || $(MAKE) tools.init
	$(BIN_DIR)/misspell -error .
	@echo "Running gosec (security)..."
	@test -f $(BIN_DIR)/gosec || $(MAKE) tools.init
	$(BIN_DIR)/gosec -fmt text ./...
	@echo "Running revive..."
	@test -f $(BIN_DIR)/revive || $(MAKE) tools.init
	$(BIN_DIR)/revive -config .revive.toml ./... || $(BIN_DIR)/revive ./...

# Testing
.PHONY: test
test:
	@test -f $(BIN_DIR)/gotestsum || $(MAKE) tools.init
	$(BIN_DIR)/gotestsum --format testname -- ./...

# E2E Testing
.PHONY: test-e2e
test-e2e: build
	@echo "Running e2e tests..."
	$(GOTEST) ./testing/e2e/... -v

.PHONY: test-e2e-quick
test-e2e-quick: build
	@echo "Running e2e tests (quick)..."
	$(GOTEST) ./testing/e2e/... -short -v

.PHONY: test-all
test-all: test test-e2e
	@echo "All tests completed"
