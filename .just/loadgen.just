# Build ev-loadgen binary
[group('loadgen')]
build-loadgen:
    @echo "--> Building ev-loadgen"
    @mkdir -p {{ build_dir }}
    @cd apps/loadgen && go build -o {{ build_dir }}/ev-loadgen .
    @echo "    Check the binary with: {{ build_dir }}/ev-loadgen"

# Build ev-loadgen Docker image
[group('loadgen')]
docker-build-loadgen:
    @echo "--> Building ev-loadgen Docker image"
    @docker build -f apps/loadgen/Dockerfile -t ev-loadgen:dev .
    @echo "--> Docker image built: ev-loadgen:dev"
