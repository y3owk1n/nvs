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
    env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version=local-build" -trimpath -o ./build/nvs-windows64.exe ./main.go

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
    env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/y3owk1n/nvs/cmd.Version={{ VERSION_OVERRIDE }}" -trimpath -o ./build/nvs-windows64.exe ./main.go

test:
    go test ./... -v

vet:
    go vet ./...

fmt:
    golangci-lint fmt
    golangci-lint run --fix

lint:
    golangci-lint run
