# tools.mk - Build configuration for ev-node tools

# Tool names
TOOLS := da-debug blob-decoder cache-analyzer

# Build directory
TOOLS_BUILD_DIR := $(CURDIR)/build

# Version information (inherited from main build.mk if available)
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
GITSHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS ?= \
	-X main.Version=$(VERSION) \
	-X main.GitSHA=$(GITSHA)

# Individual tool build targets
## build-tool-da-debug: Build da-debug tool
build-tool-da-debug:
	@echo "--> Building da-debug tool"
	@mkdir -p $(TOOLS_BUILD_DIR)
	@cd tools/da-debug && go build -ldflags "$(LDFLAGS)" -o $(TOOLS_BUILD_DIR)/da-debug .
	@echo "--> da-debug built: $(TOOLS_BUILD_DIR)/da-debug"
.PHONY: build-tool-da-debug

## build-tool-blob-decoder: Build blob-decoder tool
build-tool-blob-decoder:
	@echo "--> Building blob-decoder tool"
	@mkdir -p $(TOOLS_BUILD_DIR)
	@cd tools/blob-decoder && go build -ldflags "$(LDFLAGS)" -o $(TOOLS_BUILD_DIR)/blob-decoder .
	@echo "--> blob-decoder built: $(TOOLS_BUILD_DIR)/blob-decoder"
.PHONY: build-tool-blob-decoder

## build-tool-cache-analyzer: Build cache-analyzer tool
build-tool-cache-analyzer:
	@echo "--> Building cache-analyzer tool"
	@mkdir -p $(TOOLS_BUILD_DIR)
	@cd tools/cache-analyzer && go build -ldflags "$(LDFLAGS)" -o $(TOOLS_BUILD_DIR)/cache-analyzer .
	@echo "--> cache-analyzer built: $(TOOLS_BUILD_DIR)/cache-analyzer"
.PHONY: build-tool-cache-analyzer

# Build all tools
## build-tools: Build all tools
build-tools: $(addprefix build-tool-, $(TOOLS))
	@echo "--> All tools built successfully!"
.PHONY: build-tools

# Install individual tools
## install-tool-da-debug: Install da-debug tool to Go bin
install-tool-da-debug:
	@echo "--> Installing da-debug tool"
	@cd tools/da-debug && go install -ldflags "$(LDFLAGS)" .
	@echo "--> da-debug installed to Go bin"
.PHONY: install-tool-da-debug

## install-tool-blob-decoder: Install blob-decoder tool to Go bin
install-tool-blob-decoder:
	@echo "--> Installing blob-decoder tool"
	@cd tools/blob-decoder && go install -ldflags "$(LDFLAGS)" .
	@echo "--> blob-decoder installed to Go bin"
.PHONY: install-tool-blob-decoder

## install-tool-cache-analyzer: Install cache-analyzer tool to Go bin
install-tool-cache-analyzer:
	@echo "--> Installing cache-analyzer tool"
	@cd tools/cache-analyzer && go install -ldflags "$(LDFLAGS)" .
	@echo "--> cache-analyzer installed to Go bin"
.PHONY: install-tool-cache-analyzer

# Install all tools
## install-tools: Install all tools to Go bin
install-tools: $(addprefix install-tool-, $(TOOLS))
	@echo "--> All tools installed successfully!"
.PHONY: install-tools

# Clean built tools
## clean-tools: Remove built tool binaries
clean-tools:
	@echo "--> Cleaning built tools"
	@rm -f $(addprefix $(TOOLS_BUILD_DIR)/, $(TOOLS))
	@echo "--> Tools cleaned"
.PHONY: clean-tools

# List available tools
## list-tools: List all available tools
list-tools:
	@echo "Available tools:"
	@for tool in $(TOOLS); do \
		echo "  - $$tool"; \
	done
.PHONY: list-tools