# Run all linters
[group('lint')]
lint: vet
    @echo "--> Running golangci-lint"
    @golangci-lint run
    @echo "--> Running markdownlint"
    @markdownlint --config .markdownlint.yaml '**/*.md'
    @echo "--> Running hadolint"
    @hadolint test/docker/mockserv.Dockerfile
    @echo "--> Running yamllint"
    @yamllint --no-warnings . -c .yamllint.yml
    @echo "--> Running goreleaser check"
    @goreleaser check
    @echo "--> Running actionlint"
    @actionlint

# Auto-fix linting issues
[group('lint')]
lint-fix:
    @echo "--> Formatting go"
    @golangci-lint run --fix
    @echo "--> Formatting markdownlint"
    @markdownlint --config .markdownlint.yaml --ignore './changelog.md' '**/*.md' -f

# Run go vet
[group('lint')]
vet:
    @echo "--> Running go vet"
    @go vet ./...
