# Optimized build targets with minimal binary size
build:
    mkdir -p build
    # Build for darwin-arm64
    env GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-darwin-arm64 ./main.go

    # Build for darwin-amd64
    env GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-darwin-amd64 ./main.go

    # Build for linux-arm64
    env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-linux-arm64 ./main.go

    # Build for linux-amd64
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-linux-amd64 ./main.go

    # Build for windows-amd64
    env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-windows-amd64.exe ./main.go

    # Build for windows-arm64
    env GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-windows-arm64.exe ./main.go

release-ci VERSION_OVERRIDE:
    mkdir -p build
    # Build for darwin-arm64
    env GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ VERSION_OVERRIDE }}" -trimpath -o ./build/nvs-darwin-arm64 ./main.go

    # Build for darwin-amd64
    env GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ VERSION_OVERRIDE }}" -trimpath -o ./build/nvs-darwin-amd64 ./main.go

    # Build for linux-arm64
    env GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ VERSION_OVERRIDE }}" -trimpath -o ./build/nvs-linux-arm64 ./main.go

    # Build for linux-amd64
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ VERSION_OVERRIDE }}" -trimpath -o ./build/nvs-linux-amd64 ./main.go

    # Build for windows-amd64
    env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ VERSION_OVERRIDE }}" -trimpath -o ./build/nvs-windows-amd64.exe ./main.go

    # Build for windows-arm64
    env GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ VERSION_OVERRIDE }}" -trimpath -o ./build/nvs-windows-arm64.exe ./main.go

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
