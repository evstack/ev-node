# Build da-debug tool
[group('tools')]
build-tool-da-debug:
    @echo "--> Building da-debug tool"
    @mkdir -p {{ build_dir }}
    @cd tools/da-debug && go build -ldflags "{{ tool_ldflags }}" -o {{ build_dir }}/da-debug .
    @echo "--> da-debug built: {{ build_dir }}/da-debug"

# Build blob-decoder tool
[group('tools')]
build-tool-blob-decoder:
    @echo "--> Building blob-decoder tool"
    @mkdir -p {{ build_dir }}
    @cd tools/blob-decoder && go build -ldflags "{{ tool_ldflags }}" -o {{ build_dir }}/blob-decoder .
    @echo "--> blob-decoder built: {{ build_dir }}/blob-decoder"

# Build cache-analyzer tool
[group('tools')]
build-tool-cache-analyzer:
    @echo "--> Building cache-analyzer tool"
    @mkdir -p {{ build_dir }}
    @cd tools/cache-analyzer && go build -ldflags "{{ tool_ldflags }}" -o {{ build_dir }}/cache-analyzer .
    @echo "--> cache-analyzer built: {{ build_dir }}/cache-analyzer"

# Build all tools
[group('tools')]
build-tools: build-tool-da-debug build-tool-blob-decoder build-tool-cache-analyzer
    @echo "--> All tools built successfully!"

# Install da-debug tool to Go bin
[group('tools')]
install-tool-da-debug:
    @echo "--> Installing da-debug tool"
    @cd tools/da-debug && go install -ldflags "{{ tool_ldflags }}" .
    @echo "--> da-debug installed to Go bin"

# Install blob-decoder tool to Go bin
[group('tools')]
install-tool-blob-decoder:
    @echo "--> Installing blob-decoder tool"
    @cd tools/blob-decoder && go install -ldflags "{{ tool_ldflags }}" .
    @echo "--> blob-decoder installed to Go bin"

# Install cache-analyzer tool to Go bin
[group('tools')]
install-tool-cache-analyzer:
    @echo "--> Installing cache-analyzer tool"
    @cd tools/cache-analyzer && go install -ldflags "{{ tool_ldflags }}" .
    @echo "--> cache-analyzer installed to Go bin"

# Install all tools to Go bin
[group('tools')]
install-tools: install-tool-da-debug install-tool-blob-decoder install-tool-cache-analyzer
    @echo "--> All tools installed successfully!"

# Remove built tool binaries
[group('tools')]
clean-tools:
    @echo "--> Cleaning built tools"
    @rm -f {{ build_dir }}/da-debug {{ build_dir }}/blob-decoder {{ build_dir }}/cache-analyzer
    @echo "--> Tools cleaned"

# List available tools
[group('tools')]
list-tools:
    @echo "Available tools:"
    @echo "  - da-debug"
    @echo "  - blob-decoder"
    @echo "  - cache-analyzer"
