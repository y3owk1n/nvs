# Development Guide

This document provides information for developers working on **nvs** (Neovim Version Switcher), a CLI tool for managing Neovim installations.

## Project Overview

NVS is a command-line tool written in Go that allows users to install, switch between, and manage different versions of Neovim. It supports stable releases, nightly builds, and specific commits.

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- (Optional) Devbox for reproducible development environment

### Getting Started

1. Clone the repository:

   ```bash
   git clone https://github.com/y3owk1n/nvs.git
   cd nvs
   ```

2. If using Devbox:

   ```bash
   devbox shell
   ```

3. Install dependencies:

   ```bash
   go mod download
   ```

## Project Structure

```text
.
├── cmd/                    # Command implementations
│   ├── *_test.go          # Unit tests for commands
│   └── *_integration_test.go  # Integration tests for commands
├── pkg/                    # Core packages
│   ├── archive/           # Archive extraction utilities
│   ├── builder/           # Neovim building from source
│   ├── helpers/           # Utility functions and helpers
│   ├── installer/         # Installation logic
│   └── releases/          # Release management and API
├── docs/                  # Documentation
├── .github/workflows/     # CI/CD pipelines
├── Justfile               # Development tasks
├── go.mod                 # Go module definition
└── main.go                # Application entry point
```

## Testing Strategy

NVS follows a clear separation between unit tests and integration tests:

### Unit Tests

- **Files**: `*_test.go` (e.g., `helpers_test.go`, `releases_test.go`)
- **Purpose**: Test individual functions and logic in isolation
- **Characteristics**:
  - Fast execution
  - No external dependencies
  - Mock external interactions
  - Focus on business logic

### Integration Tests

- **Files**: `*_integration_test.go` (e.g., `cmd_integration_test.go`, `builder_integration_test.go`)
- **Purpose**: Test interactions with external systems and end-to-end functionality
- **Characteristics**:
  - Slower execution
  - May interact with file system, network, or external processes
  - Use build tag `integration`
  - Test complete workflows

### Running Tests

Use the provided Justfile recipes:

```bash
# Run all tests (unit + integration)
just test

# Run only unit tests
just test-unit

# Run only integration tests
just test-integration

# Run tests with coverage
just test-coverage

# View coverage in browser
just test-coverage-html
```

### Test Organization

- Unit tests are colocated with the code they test (e.g., `helpers.go` and `helpers_test.go`)
- Integration tests are in separate files with `_integration_test.go` suffix
- Integration tests use the `//go:build integration` build constraint

## Code Quality

### Linting and Formatting

```bash
# Run linter
just lint

# Format code and fix issues
just fmt
```

The project uses [golangci-lint](https://golangci-lint.run/) with configuration in `.golangci.yml`.

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `go vet` for static analysis
- Ensure all tests pass before committing

## Building

### Local Builds

```bash
# Build for current platform
just build

# Build release binaries for all platforms
just release-ci VERSION_OVERRIDE=v1.0.0
```

### Cross-Platform Builds

The Justfile supports building for multiple platforms:

- macOS (amd64, arm64)
- Linux (amd64, arm64)
- Windows (amd64)

## CI/CD

The project uses GitHub Actions for continuous integration:

- **CI Workflow** (`.github/workflows/ci.yml`):
  - Runs on pull requests and pushes to main
  - Tests on multiple Go versions and platforms
  - Runs linting and tests
  - Uploads test coverage

- **Release Workflow** (`.github/workflows/release-please.yml`):
  - Automated releases using release-please
  - Builds and publishes binaries on tag creation

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass: `just test`
6. Run linting: `just lint`
7. Format code: `just fmt`
8. Commit your changes
9. Push to your fork
10. Create a pull request

### Commit Messages

Follow conventional commit format:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation
- `test:` for test changes
- `refactor:` for code refactoring

### Pull Request Guidelines

- Provide a clear description of changes
- Reference any related issues
- Ensure CI passes
- Request review from maintainers

## Development Tools

### Justfile

The `Justfile` contains common development tasks. Run `just --list` to see all available recipes.

### Devbox

For a reproducible development environment, use [Devbox](https://www.jetpack.io/devbox):

```bash
devbox shell
```

This provides the exact versions of tools needed for development.

### Go Modules

Dependencies are managed with Go modules. Update dependencies with:

```bash
go get -u ./...
go mod tidy
```

## Debugging

- Use `go run main.go` for quick testing
- Enable verbose logging with `-v` flag
- Use `go test -v` for detailed test output
- Check logs with `tail -f /tmp/nvs.log` (or equivalent)

## Performance

- Unit tests should be fast (< 1s total)
- Integration tests may take longer but should complete within reasonable time
- Profile with `go test -bench=.`
- Use `go build -ldflags="-s -w"` for optimized binaries

## Security

- Never commit secrets or keys
- Use environment variables for configuration
- Validate all inputs
- Keep dependencies updated
- Run security scans regularly

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Cobra CLI Framework](https://cobra.dev/)
- [Viper Configuration](https://github.com/spf13/viper)
- [Neovim Releases API](https://github.com/neovim/neovim/releases)

## Related Documentation

- [Installation Guide](INSTALLATION.md) - How to install nvs
- [Usage Guide](USAGE.md) - Command reference and examples
- [Configuration Guide](CONFIGURATION.md) - Environment setup
- [Contributing Guide](CONTRIBUTING.md) - How to contribute
