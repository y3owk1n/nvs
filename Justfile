# Cross-compile every release target into ./build with a local version string
build: (release-ci "local-build")

# Cross-compile every release target into ./build stamped with VERSION_OVERRIDE
release-ci VERSION_OVERRIDE: \
    (build-target "darwin" "arm64" VERSION_OVERRIDE) \
    (build-target "darwin" "amd64" VERSION_OVERRIDE) \
    (build-target "linux" "arm64" VERSION_OVERRIDE) \
    (build-target "linux" "amd64" VERSION_OVERRIDE) \
    (build-target "windows" "amd64" VERSION_OVERRIDE) \
    (build-target "windows" "arm64" VERSION_OVERRIDE)

build-target os arch version:
    mkdir -p build
    env GOOS={{ os }} GOARCH={{ arch }} CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ version }}" -trimpath -o ./build/nvs-{{ os }}-{{ arch }}{{ if os == "windows" { ".exe" } else { "" } }} ./main.go

test: test-unit test-integration

test-unit:
    go test ./... -v

test-integration:
    go test -tags=integration ./... -v

test-race: test-race-unit test-race-integration

test-race-unit:
    go test -race ./... -v

test-race-integration:
    go test -tags=integration -race ./... -v

test-coverage:
    go test -coverprofile=coverage.txt ./...

test-coverage-all:
    go test -tags=integration -coverprofile=coverage-all.txt ./...

test-coverage-html:
    just test-coverage
    go tool cover -html=coverage.txt -o coverage.html

test-coverage-all-html:
    just test-coverage-all
    go tool cover -html=coverage-all.txt -o coverage-all.html

test-all: test test-race

vet:
    go vet ./...

fmt:
    golangci-lint fmt
    golangci-lint run --fix

lint:
    golangci-lint run

# Point flake, package.nix and install scripts at VERSION using hashes from ./build
update-release-refs VERSION:
    scripts/update-release-refs.sh {{ VERSION }}

vuln:
    govulncheck ./...
