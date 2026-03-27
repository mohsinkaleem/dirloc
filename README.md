# dirloc

[![CI](https://github.com/mohsinkaleem/dirloc/actions/workflows/ci.yml/badge.svg)](https://github.com/mohsinkaleem/dirloc/actions/workflows/ci.yml)
[![Release](https://github.com/mohsinkaleem/dirloc/actions/workflows/release.yml/badge.svg)](https://github.com/mohsinkaleem/dirloc/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A fast CLI tool that recursively scans a directory tree, counts lines of code, and reports top files/directories by size.
<p align="center">
  <img src="https://raw.githubusercontent.com/mohsinkaleem/dirloc/main/.github/dirloc-view.png" width="70%"/>
</p>

## Install

### Homebrew (macOS & Linux)

```bash
brew install mohsinkaleem/tap/dirloc
```

### Download Binary

Download a prebuilt binary from the [Releases](https://github.com/mohsinkaleem/dirloc/releases) page.

### From Source

```bash
go install github.com/dirloc/dirloc@latest
```

## Usage

```bash
dirloc [path] [flags]
```

### Examples

```bash
# Scan current directory
dirloc

# Language breakdown
dirloc ~/projects/myapp --lang

# Top 20 files, JSON output
dirloc . -k 20 --sort total --json

# Respect .gitignore
dirloc . --gitignore

# Cache results for faster re-scans
dirloc . --lang --cache

# Exclude specific files & dirs
dirloc . --exclude-dir test,tmp --exclude-file config.json

# Markdown report
dirloc . --lang --md > report.md

# Profiling
dirloc ~/bigproject --cpuprofile cpu.prof
go tool pprof -http=:6060 cpu.prof
```



### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--top-k` | `-k` | `10` | Number of top files/dirs to display |
| `--exclude-dir` | `-e` | — | Additional directory names to ignore |
| `--exclude-ext` | — | — | Additional file extensions to ignore |
| `--exclude-file` | — | — | Additional file names to ignore |
| `--workers` | `-w` | `NumCPU` | Parallel worker goroutines |
| `--lang` | `-l` | `false` | Show language breakdown (code/comment/blank) |
| `--complexity` | `-c` | `false` | Show complexity column |
| `--json` | — | `false` | Output as JSON |
| `--md` | — | `false` | Output as Markdown |
| `--no-top-files` | — | `false` | Suppress top files list |
| `--no-top-dirs` | — | `false` | Suppress top dirs list |
| `--sort` | `-s` | `code` | Sort by: `code`, `total`, `files` |
| `--max-file-size` | — | `10MB` | Skip files larger than this |
| `--gitignore` | — | `false` | Respect `.gitignore` files |
| `--cache` | — | `false` | Cache results in `.dirlocache` |
| `--no-progress` | — | `false` | Disable live progress indicator |
| `--cpuprofile` | — | — | Write CPU profile to file |
| `--memprofile` | — | — | Write heap profile to file |

## Features

### Two scan modes

- **Fast mode** (default) — counts total lines only via byte-level newline counting.
- **Lang mode** (`--lang`) — classifies each line as code, comment, or blank.

Both modes use the full worker pool (`--workers`) for parallel file processing.

### .gitignore support

Pass `--gitignore` to honour `.gitignore` files found during traversal. Patterns are applied per-directory, including `**` globs and `!` negation.

### File-hash cache

Pass `--cache` to store results in a `.dirlocache` file at the scan root. On subsequent runs, unchanged files (same mtime + size) are served from cache instead of being re-read.

### Progress indicator

A live `Scanning... N files [elapsed]` line is shown on stderr when the output is a terminal. Disable with `--no-progress`.

## Built-in ignores

**Directories:** `.git`, `node_modules`, `vendor`, `dist`, `build`, `.venv`, `__pycache__`, `target`, `bin`, `obj`, and more.

**Extensions:** `.exe`, `.dll`, `.png`, `.jpg`, `.zip`, `.lock`, `.sum`, `.min.js`, `.min.css`, and more.

**Files:** `package-lock.json`, `pnpm-lock.yaml`, `.DS_Store`, `.dirlocache`.

## Supported languages

120+ file extensions mapped to 60+ languages including Go, Python, JavaScript, TypeScript, Java, C, C++, Rust, Ruby, PHP, Swift, Kotlin, and many more.

## Testing

```bash
make test          # full suite
make test-short    # skip stress tests
make bench         # benchmarks with allocations
```

## Building from Source

```bash
git clone https://github.com/mohsinkaleem/dirloc.git
cd dirloc
make build
```

## Releasing a New Version

Releases are fully automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions.

1. Tag a new version:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
2. The [Release workflow](.github/workflows/release.yml) will:
   - Build binaries for Linux, macOS, and Windows (amd64 + arm64)
   - Create a GitHub Release with checksums and changelog
   - Update the Homebrew formula in [mohsinkaleem/homebrew-tap](https://github.com/mohsinkaleem/homebrew-tap)

## Architecture

```
dirloc/
├── main.go            # Entry point — embeds languages.json, calls cmd.Execute()
├── cmd/root.go        # Cobra CLI setup, flag parsing, orchestrates scan pipeline
├── scanner/
│   ├── walker.go      # Concurrent directory tree walker
│   ├── counter.go     # Line counting (fast byte-scan & language-aware modes)
│   ├── language.go    # Language detection from file extensions
│   ├── gitignore.go   # .gitignore pattern matching
│   ├── cache.go       # File-hash based scan result caching
│   ├── ignore.go      # Built-in ignore rules (dirs, extensions, files)
│   └── progress.go    # Live terminal progress indicator
├── aggregator/        # Aggregates per-file results into dir & language summaries
├── output/            # Renders results as table, JSON, or Markdown
├── types/types.go     # Shared data structures
└── languages.json     # Extension → language mapping (120+ extensions)
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
