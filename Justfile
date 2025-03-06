build:
	mkdir -p build
	# Build for darwin-arm64.
	env GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/y3owk1n/nvsw.Version=local-build" -o ./build/nvsw-darwin-arm64 ./main.go
	# Build for darwin-amd64.
	env GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/y3owk1n/nvsw.Version=local-build" -o ./build/nvsw-darwin-amd64 ./main.go
	# Build for linux-amd64.
	env GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/y3owk1n/nvsw.Version=local-build" -o ./build/nvsw-linux-amd64 ./main.go
	# Build for windows-amd64.
	env GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/y3owk1n/nvsw.Version=local-build" -o ./build/nvsw-windows-amd64.exe ./main.go
