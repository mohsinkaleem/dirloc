package scanner

import (
	"testing"
)

// --- IgnoreRules tests ---

func TestNewIgnoreRules_DefaultDirs(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, nil)

	defaults := []string{".git", "node_modules", "__pycache__", "vendor", ".vscode"}
	for _, d := range defaults {
		if !ir.ShouldSkipDir(d) {
			t.Errorf("expected %q to be skipped by default", d)
		}
	}
}

func TestNewIgnoreRules_ExtraDirs(t *testing.T) {
	ir := NewIgnoreRules([]string{"custom_dir"}, nil, nil)

	if !ir.ShouldSkipDir("custom_dir") {
		t.Error("expected custom_dir to be skipped")
	}
	if !ir.ShouldSkipDir(".git") {
		t.Error("defaults should still work with extras")
	}
}

func TestShouldSkipFile_SimpleExtension(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, nil)

	tests := []struct {
		name string
		want bool
	}{
		{"file.exe", true},
		{"file.png", true},
		{"file.zip", true},
		{"file.lock", true},
		{"file.go", false},
		{"file.py", false},
	}

	for _, tt := range tests {
		got := ir.ShouldSkipFile(tt.name)
		if got != tt.want {
			t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestShouldSkipFile_CompoundExtension(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, nil)

	// .min.js and .min.css should be skipped
	if !ir.ShouldSkipFile("jquery.min.js") {
		t.Error("expected jquery.min.js to be skipped (compound extension)")
	}
	if !ir.ShouldSkipFile("styles.min.css") {
		t.Error("expected styles.min.css to be skipped (compound extension)")
	}
	// Regular .js and .css should NOT be skipped
	if ir.ShouldSkipFile("app.js") {
		t.Error("app.js should not be skipped")
	}
}

func TestShouldSkipFile_ExtraExtensions(t *testing.T) {
	ir := NewIgnoreRules(nil, []string{"log", ".bak"}, nil)

	if !ir.ShouldSkipFile("app.log") {
		t.Error("expected .log to be skipped (extra extension)")
	}
	if !ir.ShouldSkipFile("file.bak") {
		t.Error("expected .bak to be skipped (extra extension)")
	}
}

func TestShouldSkipFile_CaseInsensitive(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, nil)

	if !ir.ShouldSkipFile("FILE.EXE") {
		t.Error("expected FILE.EXE to be skipped (case insensitive)")
	}
	if !ir.ShouldSkipFile("image.PNG") {
		t.Error("expected image.PNG to be skipped (case insensitive)")
	}
}

func TestShouldSkipFile_BuiltInFiles(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, nil)

	if !ir.ShouldSkipFile("package-lock.json") {
		t.Error("expected package-lock.json to be skipped by default")
	}
	if !ir.ShouldSkipFile("pnpm-lock.yaml") {
		t.Error("expected pnpm-lock.yaml to be skipped by default")
	}
	if !ir.ShouldSkipFile(".DS_Store") {
		t.Error("expected .DS_Store to be skipped by default")
	}
	if !ir.ShouldSkipFile(".dirlocache") {
		t.Error("expected .dirlocache to be skipped by default")
	}
}

func TestShouldSkipFile_ExtraFiles(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, []string{"config.json", "secrets.yaml"})

	if !ir.ShouldSkipFile("config.json") {
		t.Error("expected config.json to be skipped (extra file)")
	}
	if !ir.ShouldSkipFile("secrets.yaml") {
		t.Error("expected secrets.yaml to be skipped (extra file)")
	}
	if ir.ShouldSkipFile("main.go") {
		t.Error("main.go should not be skipped")
	}
}

func TestShouldSkipFile_GlobPatterns(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, []string{"*_test.go", "test_*.py"})

	tests := []struct {
		name string
		want bool
	}{
		{"main_test.go", true},
		{"handler_test.go", true},
		{"test_utils.py", true},
		{"test_main.py", true},
		{"main.go", false},
		{"utils.py", false},
	}

	for _, tt := range tests {
		got := ir.ShouldSkipFile(tt.name)
		if got != tt.want {
			t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// --- ShouldSkipLang tests ---

func TestShouldSkipLang_DocsExcludedByDefault(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, nil)
	if !ir.ShouldSkipLang("README.md", "Markdown") {
		t.Error("Markdown should be skipped by default")
	}
	if ir.ShouldSkipLang("main.go", "Go") {
		t.Error("Go should not be skipped")
	}

	ir.SetIncludes(nil, nil, true)
	if ir.ShouldSkipLang("README.md", "Markdown") {
		t.Error("Markdown should be counted with includeDocs")
	}
}

func TestShouldSkipLang_IncludeFilters(t *testing.T) {
	ir := NewIgnoreRules(nil, nil, nil)
	ir.SetIncludes([]string{"GO", ".json"}, nil, false)

	tests := []struct {
		name, lang string
		want       bool
	}{
		{"main.go", "Go", false},
		{"data.json", "JSON", false}, // explicit include overrides docs exclusion
		{"app.py", "Python", true},
	}
	for _, tt := range tests {
		if got := ir.ShouldSkipLang(tt.name, tt.lang); got != tt.want {
			t.Errorf("ShouldSkipLang(%q, %q) = %v, want %v", tt.name, tt.lang, got, tt.want)
		}
	}

	ir.SetIncludes(nil, []string{"python"}, false)
	if ir.ShouldSkipLang("app.py", "Python") {
		t.Error("--include-lang should be case-insensitive")
	}
	if !ir.ShouldSkipLang("main.go", "Go") {
		t.Error("Go should be skipped when only Python is included")
	}
}

func BenchmarkShouldSkipFile(b *testing.B) {
	ir := NewIgnoreRules(nil, nil, nil)
	names := []string{"main.go", "file.exe", "jquery.min.js", "app.py", "data.zip"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir.ShouldSkipFile(names[i%len(names)])
	}
}

func BenchmarkShouldSkipDir(b *testing.B) {
	ir := NewIgnoreRules(nil, nil, nil)
	names := []string{"src", ".git", "node_modules", "lib", "vendor"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir.ShouldSkipDir(names[i%len(names)])
	}
}
