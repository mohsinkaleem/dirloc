package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// IgnoreRules determines which directories and files to skip.
type IgnoreRules struct {
	dirs     map[string]bool
	exts     map[string]bool
	patterns []string // glob patterns from .dirlocignore
}

var defaultIgnoreDirs = []string{
	".git", ".hg", ".svn",
	"node_modules", ".venv", "venv", "__pycache__",
	".tox", ".mypy_cache", ".pytest_cache",
	"vendor", "dist", "build", ".next", ".nuxt",
	".gradle", ".idea", ".vscode", ".DS_Store",
	"target", "bin", "obj", ".terraform",
	".cache", ".eggs", ".bundle", "coverage",
	".angular", ".sass-cache",
}

var defaultIgnoreExts = []string{
	".exe", ".dll", ".so", ".dylib", ".bin", ".o", ".a", ".lib",
	".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt",
	".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".ico", ".webp",
	".mp3", ".mp4", ".avi", ".mov", ".wav", ".flac",
	".zip", ".tar", ".gz", ".rar", ".7z", ".bz2", ".xz",
	".jar", ".war", ".class", ".pyc", ".pyo",
	".woff", ".woff2", ".ttf", ".eot",
	".lock", ".sum",
	".min.js", ".min.css",
	".db", ".sqlite", ".sqlite3",
}

// NewIgnoreRules creates an IgnoreRules with built-in defaults plus extras.
func NewIgnoreRules(extraDirs, extraExts []string) *IgnoreRules {
	ir := &IgnoreRules{
		dirs: make(map[string]bool),
		exts: make(map[string]bool),
	}

	for _, d := range defaultIgnoreDirs {
		ir.dirs[d] = true
	}
	for _, d := range extraDirs {
		ir.dirs[d] = true
	}

	for _, e := range defaultIgnoreExts {
		ir.exts[strings.ToLower(e)] = true
	}
	for _, e := range extraExts {
		ext := e
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ir.exts[strings.ToLower(ext)] = true
	}

	return ir
}

// LoadIgnoreFile reads a .dirlocignore file and adds its rules.
// Each non-empty, non-comment line is interpreted as:
//   - a directory name (e.g. "vendor", "tmp")
//   - an extension pattern starting with *. (e.g. "*.log")
//   - a glob pattern for file matching (e.g. "test_*", "*.generated.*")
//
// Lines starting with # are comments.
func (ir *IgnoreRules) LoadIgnoreFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "*."):
			// Extension pattern: "*.log" → ".log"
			ext := strings.ToLower(line[1:]) // strip the "*"
			ir.exts[ext] = true
		case !strings.ContainsAny(line, "*?["):
			// Plain name → treat as directory to skip
			ir.dirs[line] = true
		default:
			// Glob pattern
			ir.patterns = append(ir.patterns, line)
		}
	}
	return s.Err()
}

// ShouldSkipDir returns true if the directory name should be skipped.
func (ir *IgnoreRules) ShouldSkipDir(name string) bool {
	return ir.dirs[name]
}

// ShouldSkipFile returns true if the file should be skipped based on extension or glob patterns.
func (ir *IgnoreRules) ShouldSkipFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ir.exts[ext] {
		return true
	}
	// Check compound extensions like .min.js, .min.css
	lowerName := strings.ToLower(name)
	for e := range ir.exts {
		if strings.Count(e, ".") > 1 && strings.HasSuffix(lowerName, e) {
			return true
		}
	}
	// Check glob patterns
	for _, p := range ir.patterns {
		if matched, _ := filepath.Match(p, name); matched {
			return true
		}
	}
	return false
}

var binaryCheckPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 512)
		return &buf
	},
}

// IsBinary reads the first 512 bytes and checks for null bytes.
func IsBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	bufPtr := binaryCheckPool.Get().(*[]byte)
	defer binaryCheckPool.Put(bufPtr)
	buf := *bufPtr

	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return false
	}

	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}
