package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// IgnoreRules determines which directories, files, and extensions to skip.
type IgnoreRules struct {
	dirs         map[string]bool
	exts         map[string]bool
	files        map[string]bool
	compoundExts []string // pre-computed compound extensions like .min.js
	fileGlobs    []string // glob patterns for file exclusion
}

var defaultIgnoreDirs = []string{
	".git", ".hg", ".svn",
	"node_modules", ".venv", "venv", "__pycache__",
	".tox", ".mypy_cache", ".pytest_cache",
	"vendor", "dist", "build", ".next", ".nuxt",
	".gradle", ".idea", ".vscode",
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

var defaultIgnoreFiles = []string{
	"package-lock.json",
	"pnpm-lock.yaml",
	".DS_Store",
	".dirlocache",
}

// NewIgnoreRules creates an IgnoreRules with built-in defaults plus extras.
func NewIgnoreRules(extraDirs, extraExts, extraFiles []string) *IgnoreRules {
	ir := &IgnoreRules{
		dirs:  make(map[string]bool),
		exts:  make(map[string]bool),
		files: make(map[string]bool),
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

	for _, f := range defaultIgnoreFiles {
		ir.files[f] = true
	}
	for _, f := range extraFiles {
		// Check if it looks like a glob pattern
		if strings.ContainsAny(f, "*?[") {
			ir.fileGlobs = append(ir.fileGlobs, f)
		} else {
			ir.files[f] = true
		}
	}

	// Pre-compute compound extensions for O(1)-ish lookup
	for e := range ir.exts {
		if strings.Count(e, ".") > 1 {
			ir.compoundExts = append(ir.compoundExts, e)
		}
	}

	return ir
}

// ShouldSkipDir returns true if the directory name should be skipped.
func (ir *IgnoreRules) ShouldSkipDir(name string) bool {
	return ir.dirs[name]
}

// ShouldSkipFile returns true if the file should be skipped based on name, extension, or glob pattern.
func (ir *IgnoreRules) ShouldSkipFile(name string) bool {
	if ir.files[name] {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ir.exts[ext] {
		return true
	}
	// Check pre-computed compound extensions like .min.js, .min.css
	if len(ir.compoundExts) > 0 {
		lowerName := strings.ToLower(name)
		for _, e := range ir.compoundExts {
			if strings.HasSuffix(lowerName, e) {
				return true
			}
		}
	}
	// Check glob patterns
	for _, pattern := range ir.fileGlobs {
		if matched, _ := filepath.Match(pattern, name); matched {
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
