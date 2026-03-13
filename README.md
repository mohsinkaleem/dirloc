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

# Use a custom exclude file
dirloc . --exclude-file .myignore

# Markdown report
dirloc . --lang --md > report.md

# Capture a CPU profile and inspect with pprof
dirloc ~/bigproject --cpuprofile cpu.prof
go tool pprof -http=:6060 cpu.prof

# Capture a heap/memory profile
dirloc ~/bigproject --memprofile mem.prof
go tool pprof -http=:6060 mem.prof
```

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--top-k` | `-k` | `10` | Number of top files/dirs to display |
| `--exclude-dir` | `-e` | — | Additional directory names to ignore |
| `--exclude-ext` | — | — | Additional file extensions to ignore |
| `--exclude-file` | — | — | Path to exclude file (default: `.dirlocignore` in scanned dir) |
| `--workers` | `-w` | `NumCPU` | Parallel worker goroutines |
| `--lang` | `-l` | `false` | Show language breakdown |
| `--complexity` | `-c` | `false` | Show complexity column |
| `--json` | — | `false` | Output as JSON |
| `--md` | — | `false` | Output as Markdown |
| `--no-top-files` | — | `false` | Suppress top files list |
| `--no-top-dirs` | — | `false` | Suppress top dirs list |
| `--sort` | `-s` | `code` | Sort by: `code`, `total`, `files` |
| `--max-file-size` | — | `10MB` | Skip files larger than this |
| `--cpuprofile` | — | — | Write CPU profile to file (use with `go tool pprof`) |
| `--memprofile` | — | — | Write heap/memory profile to file (use with `go tool pprof`) |
| `--version` | `-v` | — | Print version |

## Exclude File (`.dirlocignore`)

Place a `.dirlocignore` file in the scanned directory to automatically exclude files and directories. Each line is one of:

- **Directory name** — e.g. `tmp`, `logs` (skips directories with that name)
- **Extension pattern** — e.g. `*.log`, `*.bak` (skips files with that extension)
- **Glob pattern** — e.g. `test_*`, `*.generated.*` (skips matching filenames)
- **Comments** — lines starting with `#` are ignored

```
# .dirlocignore example
tmp
logs
generated
*.log
*.bak
test_*
```

## Profiling

`dirloc` has built-in support for Go's `pprof` profiler, which lets you find CPU hot-spots and memory allocation pressure in your own codebases — or in `dirloc` itself.

### CPU profile

```bash
# Record while scanning
dirloc ~/myproject --cpuprofile cpu.prof

# Interactive flamegraph in browser
go tool pprof -http=:6060 cpu.prof

# Quick top-10 functions in terminal
go tool pprof -top cpu.prof
```

### Memory (heap) profile

```bash
dirloc ~/myproject --memprofile mem.prof

go tool pprof -http=:6060 mem.prof
# Useful views: -alloc_space (total allocations), -inuse_space (live heap)
go tool pprof -alloc_space -top mem.prof
```

### Both profiles in one pass

```bash
dirloc ~/myproject --cpuprofile cpu.prof --memprofile mem.prof
```

Using `make`:

```bash
# Scan the current directory and open the CPU profile in a browser
make profile-cpu

# Scan a specific path
make profile-cpu PROFILE_PATH=~/myproject

# Both profiles
make profile-all PROFILE_PATH=~/myproject
```

## Testing & Benchmarks

```bash
# Full test suite
make test

# Skip stress tests (faster; good for CI)
make test-short

# Run stress tests only (creates large synthetic repos)
make stress

# All benchmarks with allocation reporting
make bench

# Targeted benchmark examples
go test ./internal/scanner/... -bench=BenchmarkFullPipeline -benchmem -benchtime=5s
go test ./internal/scanner/... -bench=BenchmarkWorkerScaling -benchmem

# Profile the benchmarks themselves
go test ./internal/scanner/... -bench=BenchmarkFullPipeline -cpuprofile bench_cpu.prof
go tool pprof -http=:6060 bench_cpu.prof
```

### Benchmark overview

| Benchmark | What it measures |
|---|---|
| `BenchmarkWalk_1KFiles` | Directory traversal speed |
| `BenchmarkProcessFiles_1KFiles` | Fast-path (total-lines) processing |
| `BenchmarkProcessFiles_1KFiles_Detailed` | Detailed (code/comment/blank) processing |
| `BenchmarkFullPipeline_10KFiles` | End-to-end: Walk + detailed scan, 10 000 files |
| `BenchmarkFullPipeline_FastPath` | End-to-end: Walk + fast scan, 10 000 files |
| `BenchmarkWorkerScaling` | Throughput vs worker count (1/2/4/8/NumCPU) |
| `BenchmarkAggregation` | Aggregation & sorting layer only |

### Stress tests

Stress tests are skipped when `-short` is passed. Run them explicitly:

```bash
go test ./internal/scanner/... -v -run=TestStress -timeout=120s
```

| Test | What it validates |
|---|---|
| `TestStress_LargeDirectoryTree` | 1 000 files across 50 dirs |
| `TestStress_DeeplyNestedDirectories` | 20 levels of nesting |
| `TestStress_FullPipeline_10K` | 10 000 files, 5 languages, full aggregate |
| `TestStress_FullPipeline_WithComplexity` | 2 000 files with complexity counting |

## Built-in Ignores

**Directories:** `.git`, `node_modules`, `vendor`, `dist`, `build`, `.venv`, `__pycache__`, `target`, `bin`, `obj`, and more.

**Extensions:** `.exe`, `.dll`, `.png`, `.jpg`, `.zip`, `.lock`, `.sum`, and more.

## Supported Languages

~120+ file extensions mapped to 60+ languages including Go, Python, JavaScript, TypeScript, Java, C, C++, Rust, Ruby, PHP, Swift, Kotlin, and many more.

## Suggested Improvements

| Area | Idea |
|---|---|
| **Parallel walking** | Replace `filepath.WalkDir` with a concurrent walker (multiple goroutines descending subdirectories) for very wide trees |
| **`.gitignore` / `.ignore` support** | Respect per-directory `.gitignore` patterns so the results match what version control sees |
| **File-hash cache** | Store a `.dirlocache` file with `(path, mtime, size) → counts` so unchanged files are not re-read on repeated scans |
| **Delta / diff reports** | Compare two scans and surface the files/dirs that grew or shrank the most |
| **Progress indicator** | Show a live counter (files scanned / elapsed) for large repos |
| **Smarter complexity** | Block-comment awareness (multi-line `/* */`, docstrings) for more accurate comment and complexity counts |
| **Top-N by language** | `--lang-top-k` flag showing the top files per language |
| **Summary-only mode** | `--summary` flag to print only the totals row without table output |
