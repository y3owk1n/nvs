# Development Guide

Technical reference for developing and maintaining **nvs**.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Project Structure](#project-structure)
- [Development Setup](#development-setup)
- [Testing](#testing)
- [Building](#building)
- [CI/CD](#cicd)
- [Debugging](#debugging)
- [Concurrency and Locking](#concurrency-and-locking)
- [Release Process](#release-process)

---

## Architecture Overview

**nvs** follows a clean architecture with clear separation of concerns:

```text
┌─────────────────────────────────────────────────┐
│                    cmd/                         │  CLI Layer
│         (Cobra commands, user interaction)      │
├─────────────────────────────────────────────────┤
│                internal/app/                    │  Application Layer
│         (Business logic, orchestration)         │
├─────────────────────────────────────────────────┤
│               internal/domain/                  │  Domain Layer
│        (Core types, interfaces, errors)         │
├─────────────────────────────────────────────────┤
│               internal/infra/                   │  Infrastructure Layer
│  (GitHub API, filesystem, archive, downloader)  │
└─────────────────────────────────────────────────┘
```

**Key Design Principles:**

- Dependency injection for testability
- Interfaces for external dependencies
- Clear separation between business logic and I/O

---

## Project Structure

```text
.
├── cmd/                        # CLI commands (Cobra)
│   ├── root.go                 # Root command, service initialization
│   ├── install.go              # nvs install
│   ├── use.go                  # nvs use
│   ├── list.go                 # nvs list
│   └── ...
├── internal/
│   ├── app/                    # Application services
│   │   ├── config/             # Configuration management
│   │   └── version/            # Version management logic
│   ├── domain/                 # Domain models & interfaces
│   │   ├── types.go            # Core types
│   │   ├── errors.go           # Domain errors
│   │   └── interfaces.go       # Repository interfaces
│   ├── infra/                  # Infrastructure implementations
│   │   ├── github/             # GitHub API client
│   │   ├── filesystem/         # File operations
│   │   ├── archive/            # Archive extraction
│   │   ├── downloader/         # HTTP downloads
│   │   ├── builder/            # Build from source
│   │   └── installer/          # Installation orchestration
│   ├── platform/               # Platform-specific code
│   └── ui/                     # User interface helpers
├── docs/                       # Documentation
├── .github/workflows/          # CI/CD pipelines
├── Justfile                    # Development tasks
├── go.mod                      # Go module
└── main.go                     # Entry point
```

---

## Development Setup

### Prerequisites

- **Go 1.21+** – [Download](https://golang.org/dl/)
- **Git** – For version control
- **Just** – Task runner ([installation](https://github.com/casey/just))
- **golangci-lint** – Linter ([installation](https://golangci-lint.run/usage/install/))

### Using Devbox (Recommended)

[Devbox](https://www.jetpack.io/devbox) provides a reproducible environment:

```bash
devbox shell
```

### Manual Setup

```bash
# Clone repository
git clone https://github.com/y3owk1n/nvs.git
cd nvs

# Install dependencies
go mod download

# Verify setup
just build
just test
```

---

## Testing

### Test Organization

| Pattern                 | Type        | Description                             |
| ----------------------- | ----------- | --------------------------------------- |
| `*_test.go`             | Unit        | Fast, isolated, mock dependencies       |
| `*_integration_test.go` | Integration | Real I/O, uses `//go:build integration` |

### Running Tests

```bash
# All tests
just test

# Unit tests only (fast)
just test-unit

# Integration tests only
just test-integration

# With coverage
just test-coverage
just test-coverage-html  # Opens in browser
```

### Writing Tests

**Unit test example:**

```go
func TestParseVersion(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Version
        wantErr bool
    }{
        {"stable", "stable", Version{Type: Stable}, false},
        {"nightly", "nightly", Version{Type: Nightly}, false},
        {"tag", "v0.10.3", Version{Type: Tag, Value: "v0.10.3"}, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseVersion(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Integration test example:**

```go
//go:build integration

package cmd_test

func TestInstallStable(t *testing.T) {
    // Uses real filesystem and network
    // Run with: just test-integration
}
```

---

## Building

### Local Build

```bash
# Current platform
just build

# Output: ./build/nvs
```

### Cross-Platform Builds

```bash
# All platforms (used in release)
just release-ci VERSION_OVERRIDE=v1.0.0

# Manual cross-compile
GOOS=darwin GOARCH=arm64 go build -o nvs-darwin-arm64 ./main.go
GOOS=linux GOARCH=amd64 go build -o nvs-linux-amd64 ./main.go
GOOS=windows GOARCH=amd64 go build -o nvs-windows-amd64.exe ./main.go
```

### Build Flags

```bash
# Optimized binary (smaller, stripped)
go build -ldflags="-s -w" -o nvs ./main.go

# With version info
go build -ldflags="-X 'cmd.Version=v1.2.3'" -o nvs ./main.go
```

---

## CI/CD

### Workflows

| Workflow             | Trigger  | Purpose                        |
| -------------------- | -------- | ------------------------------ |
| `ci.yml`             | Push, PR | Lint, test, build verification |
| `release-please.yml` | Tag      | Automated releases             |

### CI Checks

On every PR:

1. **Lint** – golangci-lint
2. **Test** – Unit + integration tests
3. **Build** – Verify compilation
4. **Coverage** – Upload to coverage service

### Local CI Simulation

```bash
# Run all CI checks locally
just fmt
just lint
just test
just build
```

---

## Debugging

### Verbose Mode

```bash
nvs --verbose install stable
nvs -v use nightly
```

### Development Run

```bash
# Run without building
go run main.go install stable
go run main.go --verbose use nightly
```

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug ./main.go -- install stable
```

### Common Debug Scenarios

**Check service initialization:**

```bash
NVS_CONFIG_DIR=/tmp/nvs-debug nvs --verbose doctor
```

**Inspect cache:**

```bash
cat ~/.cache/nvs/releases.json | jq
```

**Test symlink creation:**

```bash
ls -la ~/.local/bin/nvim
```

---

## Release Process

### Automated (Preferred)

Releases are managed by [release-please](https://github.com/google-github-actions/release-please-action):

1. Merge PRs with conventional commits
2. Release-please creates a release PR
3. Merge the release PR to create a GitHub release
4. CI builds and publishes binaries

### Manual Release

```bash
# Tag the release
git tag v1.2.3
git push origin v1.2.3

# CI will build and publish
```

### Version Bumping

Based on conventional commits:

- `fix:` → Patch (1.0.0 → 1.0.1)
- `feat:` → Minor (1.0.0 → 1.1.0)
- `feat!:` or `BREAKING CHANGE:` → Major (1.0.0 → 2.0.0)

---

## Dependencies

### Key Dependencies

| Package                                      | Purpose            |
| -------------------------------------------- | ------------------ |
| [cobra](https://cobra.dev/)                  | CLI framework      |
| [logrus](https://github.com/sirupsen/logrus) | Structured logging |

### Updating Dependencies

```bash
go get -u ./...
go mod tidy
```

---

## Performance

### Guidelines

- Unit tests should complete in < 1s total
- Keep binary size reasonable (< 15MB)
- Minimize network calls (use caching)

### Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

---

## Concurrency and Locking

**nvs** uses file-based locking to ensure thread-safe and process-safe concurrent operations on version storage.

### Why File Locking?

Multiple `nvs` processes may run concurrently (e.g., parallel installs, simultaneous `use` and `uninstall` operations). Without coordination, race conditions can corrupt:
- Symlinks (concurrent `Switch` operations)
- Version directories (concurrent `Install`/`Uninstall`)
- Build artifacts (concurrent `BuildFromCommit`)

### Lock Implementation

Uses cross-platform advisory file locking:
- **Unix**: `flock()` system call
- **Windows**: `LockFileEx` API

Lock files are **not deleted** after use to prevent inode reuse race conditions.

### Lock Strategy

**Per-version locking**: Each version has its own lock file:
```
{versions_dir}/.nvs-version-{version_name}.lock
```

This allows:
- Concurrent operations on **different** versions
- Exclusive access for operations on the **same** version

### Protected Operations

| Operation | Lock File | Purpose |
|-----------|-----------|---------|
| `Switch` | `.nvs-version-{version}.lock` | Prevent concurrent symlink updates |
| `Install` | `.nvs-version-{version}.lock` | Prevent concurrent installs of same version |
| `Uninstall` | `.nvs-version-{version}.lock` | Coordinate with Switch/Install |
| `BuildFromCommit` | `.nvs-version-{commit}.lock` | Prevent concurrent builds of same commit |

### Example Race Scenarios Prevented

**Scenario 1: Switch + Uninstall**
```
Process A: nvs use v1.0.0    → acquires lock for v1.0.0
Process B: nvs uninstall v1.0.0 → waits for lock
Process A: updates symlinks → releases lock
Process B: acquires lock → checks if current → removes directory
```

**Scenario 2: Concurrent Install**
```
Process A: nvs install v1.0.0 → acquires lock for v1.0.0
Process B: nvs install v1.0.0 → waits for lock
Process A: downloads/extracts → releases lock
Process B: acquires lock → sees version already exists
```

### Timeout Behavior

**Standard operations** (Switch, Uninstall, Install): **30 seconds**

**Build operations** (BuildFromCommit): **15 minutes**
  - Builds can take several minutes with auto-retry logic
  - Extended timeout prevents premature failures during long builds

If a lock cannot be acquired within the timeout, the operation fails with:
```
timeout waiting for file lock
```

### Fast-Path Optimization

Both `Install` and `BuildFromCommit` implement a fast-path check:

1. **Before locking**: Check if version already exists
2. **If exists**: Return immediately without acquiring lock
3. **After locking**: Double-check (another process may have installed)

This prevents unnecessary waiting when:
- Version is already installed
- Another process completes installation while waiting

### Production Considerations

**Build Retry + Locking:**
- Build operations auto-retry up to 3 times on failure
- Lock is held across all retry attempts
- Extended 15-minute timeout accommodates retry loops

**Example Build Scenario:**
```
Process A: nvs install --from-source abc123
  → acquires lock
  → attempt 1 fails (network timeout)
  → waits 1 second
  → attempt 2 fails (build error)
  → waits 1 second
  → attempt 3 succeeds
  → releases lock (total time: 8 minutes)

Process B: nvs install --from-source abc123 (starts 2 min after A)
  → sees version exists (fast-path)
  → returns immediately (no lock needed)
```

### Debugging Lock Issues

```bash
# Check for stale lock files
ls -la ~/.local/share/nvs/versions/.nvs-version-*.lock

# Force remove (use with caution - may crash active process)
rm ~/.local/share/nvs/versions/.nvs-version-v1.0.0.lock
```

---

## Security

- Never commit secrets or API keys
- Validate all user inputs
- Use HTTPS for all network calls
- Keep dependencies updated
- Run `go mod tidy` regularly

---

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Cobra CLI Framework](https://cobra.dev/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Neovim Releases API](https://api.github.com/repos/neovim/neovim/releases)
