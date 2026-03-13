# dirloc

A fast CLI tool that recursively scans a directory tree, counts lines of code, and reports top files/directories by size.

## Install

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
