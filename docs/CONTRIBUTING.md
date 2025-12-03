# Contributing Guide

We welcome contributions to **nvs**! This guide explains how to get involved in the project.

## Ways to Contribute

- **Bug Reports:** Report issues via [GitHub Issues](https://github.com/y3owk1n/nvs/issues)
- **Feature Requests:** Suggest new features or improvements
- **Code Contributions:** Submit pull requests with fixes or enhancements
- **Documentation:** Improve docs, tutorials, or examples
- **Testing:** Help test on different platforms or report compatibility issues

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- (Optional) Devbox for reproducible environment

### Getting Started

1. **Fork the repository** on GitHub

2. **Clone your fork:**

   ```bash
   git clone https://github.com/your-username/nvs.git
   cd nvs
   ```

3. **Set up development environment:**

   ```bash
   # If using Devbox
   devbox shell

   # Install dependencies
   go mod download
   ```

4. **Create a feature branch:**

   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

### Code Changes

1. **Make your changes** following the coding standards
2. **Add tests** for new functionality
3. **Run tests** to ensure everything works:

   ```bash
   # Run all tests
   just test

   # Run only unit tests
   just test-unit

   # Run integration tests
   just test-integration
   ```

4. **Run linting:**

   ```bash
   just lint
   ```

5. **Format code:**

   ```bash
   just fmt
   ```

### Testing Strategy

**nvs** uses a clear separation between unit and integration tests:

- **Unit Tests:** Test individual functions in isolation
- **Integration Tests:** Test end-to-end functionality with build tag `integration`

### Commit Guidelines

Follow conventional commit format:

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Test additions/modifications
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

Examples:

```
feat: add support for nightly builds
fix: resolve symlink creation on Windows
docs: update installation instructions
test: add integration tests for config switching
```

### Pull Request Process

1. **Ensure all tests pass** and linting is clean
2. **Update documentation** if needed
3. **Write clear commit messages**
4. **Push to your fork:**

   ```bash
   git push origin feature/your-feature-name
   ```

5. **Create a Pull Request** on GitHub with:
   - Clear title and description
   - Reference any related issues
   - Screenshots/demos for UI changes

## Code Standards

### Go Conventions

- Follow standard Go formatting (`gofmt`)
- Use `go vet` for static analysis
- Keep functions focused and testable
- Add comments for exported functions
- Handle errors appropriately

### Project Structure

```
.
├── cmd/                    # CLI commands
│   ├── *_test.go          # Unit tests
│   └── *_integration_test.go  # Integration tests
├── pkg/                    # Core packages
│   ├── archive/           # Archive handling
│   ├── builder/           # Neovim building
│   ├── helpers/           # Utilities
│   ├── installer/         # Installation logic
│   └── releases/          # Release management
├── docs/                  # Documentation
└── main.go                # Entry point
```

### Testing Guidelines

- Write unit tests for all new functions
- Use table-driven tests where appropriate
- Mock external dependencies in unit tests
- Integration tests should use real dependencies
- Aim for good test coverage

## Platform Support

**nvs** aims to work on:

- macOS (Intel & Apple Silicon)
- Linux (various distributions)
- Windows (limited support)

When contributing:

- Test on your platform
- Consider cross-platform compatibility
- Report platform-specific issues

## Issue Reporting

When reporting bugs:

1. **Check existing issues** first
2. **Use issue templates** when available
3. **Provide system information:**
   - OS and version
   - Go version
   - **nvs** version
   - Steps to reproduce
4. **Include error messages** and logs (use `--verbose`)
5. **Attach relevant files** if needed

## Documentation

- Keep README minimal and link to detailed docs
- Update docs for any new features
- Ensure examples are accurate and tested
- Use clear, concise language

## Community Guidelines

- Be respectful and constructive
- Help newcomers get started
- Focus on solutions, not blame

## Getting Help

- **Documentation:** Check [docs/](docs/) first
- **Issues:** Search existing GitHub issues
- **Discussions:** Use GitHub Discussions for questions
- **Discord/Slack:** Check project communication channels

## Related Documentation

- [Installation Guide](INSTALLATION.md) - How to install nvs
- [Usage Guide](USAGE.md) - Command reference
- [Configuration Guide](CONFIGURATION.md) - Environment setup
- [Development Guide](DEVELOPMENT.md) - Development setup

## Recognition

Contributors are recognized in:

- GitHub's contributor insights
- CHANGELOG.md for significant contributions
- Release notes

Thank you for contributing to **nvs**! 🚀
