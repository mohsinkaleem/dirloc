package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// --- IgnoreRules tests ---

func TestNewIgnoreRules_DefaultDirs(t *testing.T) {
	ir := NewIgnoreRules(nil, nil)

	defaults := []string{".git", "node_modules", "__pycache__", "vendor", ".vscode"}
	for _, d := range defaults {
		if !ir.ShouldSkipDir(d) {
			t.Errorf("expected %q to be skipped by default", d)
		}
	}
}

func TestNewIgnoreRules_ExtraDirs(t *testing.T) {
	ir := NewIgnoreRules([]string{"custom_dir"}, nil)

	if !ir.ShouldSkipDir("custom_dir") {
		t.Error("expected custom_dir to be skipped")
	}
	if !ir.ShouldSkipDir(".git") {
		t.Error("defaults should still work with extras")
	}
}

func TestShouldSkipFile_SimpleExtension(t *testing.T) {
	ir := NewIgnoreRules(nil, nil)

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
	ir := NewIgnoreRules(nil, nil)

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
	ir := NewIgnoreRules(nil, []string{"log", ".bak"})

	if !ir.ShouldSkipFile("app.log") {
		t.Error("expected .log to be skipped (extra extension)")
	}
	if !ir.ShouldSkipFile("file.bak") {
		t.Error("expected .bak to be skipped (extra extension)")
	}
}

func TestShouldSkipFile_CaseInsensitive(t *testing.T) {
	ir := NewIgnoreRules(nil, nil)

	if !ir.ShouldSkipFile("FILE.EXE") {
		t.Error("expected FILE.EXE to be skipped (case insensitive)")
	}
	if !ir.ShouldSkipFile("image.PNG") {
		t.Error("expected image.PNG to be skipped (case insensitive)")
	}
}

// --- IsBinary tests ---

func TestIsBinary_TextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "text.txt")
	os.WriteFile(path, []byte("hello world\n"), 0644)

	if IsBinary(path) {
		t.Error("text file should not be detected as binary")
	}
}

func TestIsBinary_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	data := make([]byte, 100)
	data[50] = 0 // null byte
	os.WriteFile(path, data, 0644)

	if !IsBinary(path) {
		t.Error("file with null byte should be detected as binary")
	}
}

func TestIsBinary_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty")
	os.WriteFile(path, []byte{}, 0644)

	if IsBinary(path) {
		t.Error("empty file should not be detected as binary")
	}
}

func TestIsBinary_NonExistent(t *testing.T) {
	if IsBinary("/nonexistent/file") {
		t.Error("nonexistent file should not be detected as binary")
	}
}

// --- LoadIgnoreFile tests ---

func TestLoadIgnoreFile_Basic(t *testing.T) {
	dir := t.TempDir()
	ignoreFile := filepath.Join(dir, ".dirlocignore")
	content := `# comment line
tmp
logs
*.log
*.bak
test_*
`
	os.WriteFile(ignoreFile, []byte(content), 0644)

	ir := NewIgnoreRules(nil, nil)
	if err := ir.LoadIgnoreFile(ignoreFile); err != nil {
		t.Fatalf("LoadIgnoreFile: %v", err)
	}

	// Directory rules
	if !ir.ShouldSkipDir("tmp") {
		t.Error("expected 'tmp' to be skipped")
	}
	if !ir.ShouldSkipDir("logs") {
		t.Error("expected 'logs' to be skipped")
	}

	// Extension rules
	if !ir.ShouldSkipFile("app.log") {
		t.Error("expected *.log to be skipped")
	}
	if !ir.ShouldSkipFile("backup.bak") {
		t.Error("expected *.bak to be skipped")
	}

	// Glob pattern rules
	if !ir.ShouldSkipFile("test_helper.go") {
		t.Error("expected test_* glob to match")
	}

	// Should not affect unrelated files
	if ir.ShouldSkipFile("main.go") {
		t.Error("main.go should not be skipped")
	}
	if ir.ShouldSkipDir("src") {
		t.Error("src should not be skipped")
	}
}

func TestLoadIgnoreFile_EmptyAndComments(t *testing.T) {
	dir := t.TempDir()
	ignoreFile := filepath.Join(dir, ".dirlocignore")
	content := `# just comments

# and blank lines

`
	os.WriteFile(ignoreFile, []byte(content), 0644)

	ir := NewIgnoreRules(nil, nil)
	if err := ir.LoadIgnoreFile(ignoreFile); err != nil {
		t.Fatalf("LoadIgnoreFile: %v", err)
	}

	// Should only have defaults
	if !ir.ShouldSkipDir(".git") {
		t.Error("defaults should still work")
	}
}

func TestLoadIgnoreFile_NonExistent(t *testing.T) {
	ir := NewIgnoreRules(nil, nil)
	err := ir.LoadIgnoreFile("/nonexistent/.dirlocignore")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func BenchmarkIsBinary_Text(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "text.txt")
	os.WriteFile(path, []byte("hello world line of text\n"), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		IsBinary(path)
	}
}

func BenchmarkShouldSkipFile(b *testing.B) {
	ir := NewIgnoreRules(nil, nil)
	names := []string{"main.go", "file.exe", "jquery.min.js", "app.py", "data.zip"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir.ShouldSkipFile(names[i%len(names)])
	}
}

func BenchmarkShouldSkipDir(b *testing.B) {
	ir := NewIgnoreRules(nil, nil)
	names := []string{"src", ".git", "node_modules", "lib", "vendor"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ir.ShouldSkipDir(names[i%len(names)])
	}
}
