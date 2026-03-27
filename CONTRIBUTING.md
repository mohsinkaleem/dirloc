# Contributing to dirloc

Thanks for your interest in contributing!

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/<your-username>/dirloc.git
   cd dirloc
   ```
3. Create a branch:
   ```bash
   git checkout -b my-feature
   ```
4. Make your changes and add tests
5. Run the test suite:
   ```bash
   make test
   make lint
   ```
6. Commit and push:
   ```bash
   git commit -m "feat: add my feature"
   git push origin my-feature
   ```
7. Open a Pull Request against `main`

## Development

### Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

### Useful Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the binary |
| `make test` | Run all tests with `-v` |
| `make test-short` | Run tests skipping stress tests |
| `make bench` | Run benchmarks with allocation stats |
| `make lint` | Run golangci-lint |
| `make clean` | Remove build artifacts |

### Project Structure

- **`cmd/`** — CLI flag parsing and scan orchestration (Cobra)
- **`scanner/`** — Directory walking, line counting, language detection, caching
- **`aggregator/`** — Aggregates per-file stats into directory and language summaries
- **`output/`** — Table, JSON, and Markdown renderers
- **`types/`** — Shared data structures

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- All exported functions should have doc comments
- Keep functions focused and small
- Add tests for new functionality

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — New feature
- `fix:` — Bug fix
- `docs:` — Documentation changes
- `test:` — Adding/updating tests
- `ci:` — CI/CD changes
- `refactor:` — Code refactoring
- `perf:` — Performance improvements

## Reporting Issues

Please use [GitHub Issues](https://github.com/mohsinkaleem/dirloc/issues) and include:

- Your OS and Go version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or output
