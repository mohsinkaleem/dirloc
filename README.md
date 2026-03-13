# dirloc

A fast CLI tool that recursively scans a directory tree, counts lines of code per file, aggregates stats by directory and language, and reports Top-K files/directories.

## Installation

```bash
go install github.com/dirloc/dirloc@latest
```

Or build from source:

```bash
make build
```

## Usage

```bash
dirloc [path] [flags]
```

### Examples

```bash
# Scan current directory
dirloc

# Scan a specific path with language breakdown
dirloc ~/projects/myapp --lang

# Top 20 files, sorted by total lines, JSON output
dirloc . --top-k 20 --sort total --json

# Exclude test directories, show complexity
dirloc . --exclude-dir test,tests --complexity

# Markdown report
dirloc . --lang --md > report.md
```

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--top-k` | `-k` | `10` | Number of top files/dirs to display |
| `--exclude-dir` | `-e` | — | Additional directory names to ignore |
| `--exclude-ext` | — | — | Additional file extensions to ignore |
| `--workers` | `-w` | `NumCPU` | Parallel worker goroutines |
| `--lang` | `-l` | `false` | Show language breakdown |
| `--complexity` | `-c` | `false` | Show complexity column |
| `--json` | — | `false` | Output as JSON |
| `--md` | — | `false` | Output as Markdown |
| `--no-top-files` | — | `false` | Suppress top files list |
| `--no-top-dirs` | — | `false` | Suppress top dirs list |
| `--sort` | `-s` | `code` | Sort by: `code`, `total`, `files` |
| `--max-file-size` | — | `10MB` | Skip files larger than this |
| `--version` | `-v` | — | Print version |

## Built-in Ignores

**Directories:** `.git`, `node_modules`, `vendor`, `dist`, `build`, `.venv`, `__pycache__`, `target`, `bin`, `obj`, and more.

**Extensions:** `.exe`, `.dll`, `.png`, `.jpg`, `.zip`, `.lock`, `.sum`, and more.

## Supported Languages

~120+ file extensions mapped to 60+ languages including Go, Python, JavaScript, TypeScript, Java, C, C++, Rust, Ruby, PHP, Swift, Kotlin, and many more.
