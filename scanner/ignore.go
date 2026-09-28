package scanner

import (
	"path/filepath"
	"strings"
)

// IgnoreRules determines which directories, files, and extensions to skip.
type IgnoreRules struct {
	dirs         map[string]bool
	exts         map[string]bool
	files        map[string]bool
	compoundExts []string // pre-computed compound extensions like .min.js
	fileGlobs    []string // glob patterns for file exclusion
	includeExts  map[string]bool
	includeLangs map[string]bool // lowercased
	includeDocs  bool
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
		ir.exts[e] = true
	}
	for _, e := range extraExts {
		ir.exts[normalizeExt(e)] = true
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

// normalizeExt turns "GO", "go" or ".go" into ".go".
func normalizeExt(e string) string {
	e = strings.ToLower(e)
	if !strings.HasPrefix(e, ".") {
		e = "." + e
	}
	return e
}

// SetIncludes limits scanning to the given extensions and languages (case-insensitive).
// Documentation/data languages are skipped unless includeDocs is set or they are
// explicitly included.
func (ir *IgnoreRules) SetIncludes(exts, langs []string, includeDocs bool) {
	ir.includeExts = make(map[string]bool, len(exts))
	for _, e := range exts {
		ir.includeExts[normalizeExt(e)] = true
	}
	ir.includeLangs = make(map[string]bool, len(langs))
	for _, l := range langs {
		ir.includeLangs[strings.ToLower(l)] = true
	}
	ir.includeDocs = includeDocs
}

// ShouldSkipLang reports whether a file with this name and detected language is filtered out.
func (ir *IgnoreRules) ShouldSkipLang(name, lang string) bool {
	if len(ir.includeLangs) > 0 && !ir.includeLangs[strings.ToLower(lang)] {
		return true
	}
	if len(ir.includeExts) > 0 && !ir.includeExts[strings.ToLower(filepath.Ext(name))] {
		return true
	}
	explicit := len(ir.includeLangs) > 0 || len(ir.includeExts) > 0
	return !explicit && !ir.includeDocs && IsDocLanguage(lang)
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
