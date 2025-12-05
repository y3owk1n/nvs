# Contributing Guide

Thank you for your interest in contributing to **nvs**! This guide covers everything you need to get started.

---

## Ways to Contribute

| Type                | Description                                                                    |
| ------------------- | ------------------------------------------------------------------------------ |
| 🐛 Bug Reports      | [Open an issue](https://github.com/y3owk1n/nvs/issues) with reproduction steps |
| 💡 Feature Requests | Suggest improvements via GitHub Issues                                         |
| 🔧 Code             | Submit pull requests for fixes or features                                     |
| 📖 Documentation    | Improve docs, examples, or tutorials                                           |
| 🧪 Testing          | Test on different platforms, report compatibility issues                       |

---

## Quick Start

```bash
# 1. Fork and clone
git clone https://github.com/YOUR-USERNAME/nvs.git
cd nvs

# 2. Set up environment (optional: use Devbox)
devbox shell  # or just ensure Go 1.21+ is installed

# 3. Install dependencies
go mod download

# 4. Verify setup
just test
just lint
just build
```

---

## Development Workflow

### Making Changes

1. **Create a branch:**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes** following the [code standards](#code-standards)

3. **Add tests** for new functionality

4. **Run quality checks:**

   ```bash
   just fmt     # Format code
   just lint    # Run linter
   just test    # Run all tests
   just build   # Verify build
   ```

5. **Commit with conventional format:**

   ```bash
   git commit -m "feat: add support for version aliases"
   ```

### Submitting a PR

1. Push to your fork:

   ```bash
   git push origin feature/your-feature-name
   ```

2. Open a Pull Request on GitHub with:
   - Clear title describing the change
   - Description of what and why
   - Reference to related issues (e.g., `Fixes #123`)
   - Screenshots for UI changes

3. Wait for CI to pass and address review feedback

---

## Code Standards

### Go Conventions

- Follow standard Go formatting (`gofmt`)
- Run `go vet` for static analysis
- Keep functions focused and testable
- Document exported functions
- Handle errors explicitly (no silent failures)

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

| Prefix      | Use Case                 |
| ----------- | ------------------------ |
| `feat:`     | New features             |
| `fix:`      | Bug fixes                |
| `docs:`     | Documentation only       |
| `test:`     | Test additions/changes   |
| `refactor:` | Code restructuring       |
| `chore:`    | Maintenance tasks        |
| `perf:`     | Performance improvements |

**Examples:**

```
feat: add support for version aliases
fix: resolve symlink creation on Windows
docs: update installation instructions for Nix
test: add integration tests for config switching
```

### Testing Guidelines

- Write tests for all new functionality
- Use table-driven tests where appropriate
- Unit tests: Fast, isolated, mock external dependencies
- Integration tests: Use `//go:build integration` tag

```bash
just test             # All tests
just test-unit        # Unit tests only
just test-integration # Integration tests only
just test-coverage    # With coverage report
```

---

## First-Time Contributors

New to the project? Look for issues labeled:

- `good first issue` – Simple, well-defined tasks
- `help wanted` – Ready for community contribution

### Your First PR

1. Pick a `good first issue`
2. Comment that you're working on it
3. Follow the [development workflow](#development-workflow)
4. Don't hesitate to ask questions!

---

## Reporting Issues

### Before Opening an Issue

1. Search existing issues to avoid duplicates
2. Check the documentation
3. Run `nvs doctor` for common problems

### Bug Reports Should Include

- **System info:** OS, architecture, shell
- **nvs version:** Output of `nvs --version`
- **Steps to reproduce:** Minimal, complete steps
- **Expected vs actual behavior**
- **Error messages:** Use `--verbose` for detailed logs

**Template:**

```markdown
**Environment:**

- OS: macOS 14.0 (arm64)
- Shell: zsh 5.9
- nvs version: v1.2.3

**Steps to reproduce:**

1. Run `nvs install nightly`
2. Run `nvs use nightly`
3. ...

**Expected:** Should switch to nightly
**Actual:** Error: <paste error>

**Verbose output:**
<paste output of `nvs --verbose use nightly`>
```

---

## Pull Request Checklist

Before submitting, ensure:

- [ ] Code follows project style
- [ ] Tests pass locally (`just test`)
- [ ] Linting passes (`just lint`)
- [ ] New features have tests
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventions
- [ ] PR description explains the change

---

## Code Review

What to expect:

- Maintainers aim to review PRs within a few days
- Feedback is meant to improve the code, not criticize you
- Discussion is encouraged – we want the best solution
- Changes may be requested before merging

---

## Community Guidelines

- Be respectful and constructive
- Help newcomers get started
- Focus on solutions, not blame
- Assume good intentions

---

## Getting Help

- **Documentation:** Check [docs/](.) first
- **Issues:** Search [GitHub Issues](https://github.com/y3owk1n/nvs/issues)
- **Discussions:** Use [GitHub Discussions](https://github.com/y3owk1n/nvs/discussions)

---

## Recognition

Contributors are recognized in:

- GitHub's contributor insights
- Release notes for significant contributions
- CHANGELOG.md

Thank you for contributing to **nvs**! 🚀
