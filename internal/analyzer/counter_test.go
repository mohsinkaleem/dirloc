package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- CountLines tests ---

func TestCountLines_SimpleGoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	content := `package main

import "fmt"

// main function
func main() {
	fmt.Println("hello")
}
`
	os.WriteFile(path, []byte(content), 0644)

	result, err := CountLines(path, "Go", []string{"//"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Language != "Go" {
		t.Errorf("language = %q, want Go", result.Language)
	}
	if result.Code != 5 {
		t.Errorf("code = %d, want 5", result.Code)
	}
	if result.Comment != 1 {
		t.Errorf("comment = %d, want 1", result.Comment)
	}
	if result.Blank != 2 {
		t.Errorf("blank = %d, want 2", result.Blank)
	}
	if result.Total != 8 {
		t.Errorf("total = %d, want 8", result.Total)
	}
}

func TestCountLines_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.go")
	os.WriteFile(path, []byte(""), 0644)

	result, err := CountLines(path, "Go", []string{"//"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
}

func TestCountLines_OnlyBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blank.go")
	os.WriteFile(path, []byte("\n\n\n"), 0644)

	result, err := CountLines(path, "Go", []string{"//"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Blank != 3 {
		t.Errorf("blank = %d, want 3", result.Blank)
	}
	if result.Code != 0 {
		t.Errorf("code = %d, want 0", result.Code)
	}
}

func TestCountLines_OnlyComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comments.py")
	os.WriteFile(path, []byte("# line1\n# line2\n# line3\n"), 0644)

	result, err := CountLines(path, "Python", []string{"#"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Comment != 3 {
		t.Errorf("comment = %d, want 3", result.Comment)
	}
	if result.Code != 0 {
		t.Errorf("code = %d, want 0", result.Code)
	}
}

func TestCountLines_Complexity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "complex.go")
	content := `if x > 0 {
	for i := range items {
		switch v {
		case 1:
		case 2:
		}
	}
} else {
	x = 0
}
`
	os.WriteFile(path, []byte(content), 0644)

	result, err := CountLines(path, "Go", []string{"//"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Complexity == 0 {
		t.Error("expected non-zero complexity")
	}
}

func TestCountLines_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noeol.go")
	os.WriteFile(path, []byte("package main"), 0644)

	result, err := CountLines(path, "Go", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
}

func TestCountLines_NonExistentFile(t *testing.T) {
	result, err := CountLines("/nonexistent/file.go", "Go", nil, false)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if result.Error == "" {
		t.Error("expected error message in result")
	}
}

func TestCountLines_MultipleCommentPrefixes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.sql")
	content := `-- SQL comment
SELECT * FROM users;
# another comment style
WHERE id = 1;
`
	os.WriteFile(path, []byte(content), 0644)

	result, err := CountLines(path, "SQL", []string{"--", "#"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Comment != 2 {
		t.Errorf("comment = %d, want 2", result.Comment)
	}
	if result.Code != 2 {
		t.Errorf("code = %d, want 2", result.Code)
	}
}

// --- CountTotalLines tests ---

func TestCountTotalLines_Simple(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0644)

	result, err := CountTotalLines(path, "Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("total = %d, want 3", result.Total)
	}
	if result.Language != "Go" {
		t.Errorf("language = %q, want Go", result.Language)
	}
}

func TestCountTotalLines_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("line1\nline2\nline3"), 0644)

	result, err := CountTotalLines(path, "Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("total = %d, want 3", result.Total)
	}
}

func TestCountTotalLines_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.go")
	os.WriteFile(path, []byte(""), 0644)

	result, err := CountTotalLines(path, "Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
}

func TestCountTotalLines_SingleNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("\n"), 0644)

	result, err := CountTotalLines(path, "Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
}

func TestCountTotalLines_LargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.go")

	// Create a file with 100k lines
	var b strings.Builder
	for i := 0; i < 100_000; i++ {
		b.WriteString("package main\n")
	}
	os.WriteFile(path, []byte(b.String()), 0644)

	result, err := CountTotalLines(path, "Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 100_000 {
		t.Errorf("total = %d, want 100000", result.Total)
	}
}

func TestCountTotalLines_ConsistentWithCountLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := `package main

import "fmt"

// hello
func main() {
	fmt.Println("hello")
}
`
	os.WriteFile(path, []byte(content), 0644)

	detailed, _ := CountLines(path, "Go", []string{"//"}, false)
	fast, _ := CountTotalLines(path, "Go")

	if detailed.Total != fast.Total {
		t.Errorf("CountLines total=%d vs CountTotalLines total=%d", detailed.Total, fast.Total)
	}
}

func TestCountTotalLines_NonExistentFile(t *testing.T) {
	result, err := CountTotalLines("/nonexistent/file.go", "Go")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if result.Error == "" {
		t.Error("expected error message in result")
	}
}

// --- countBranchKeywords tests ---

func TestCountBranchKeywords(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{"if x > 0 {", 1},
		{"for i := range items {", 1},
		{"while (true) {", 1},
		{"switch v {", 1},
		{"case 1:", 1},
		{"else {", 1},
		{"elif x:", 1},
		{"catch (err) {", 1},
		{"except ValueError:", 1},
		{"x = 1", 0},
		{"fmt.Println(x)", 0},
		{"if x > 0 { for i := 0; i < 10; i++ {", 2},
		{"", 0},
	}

	for _, tt := range tests {
		got := countBranchKeywords(tt.line)
		if got != tt.expected {
			t.Errorf("countBranchKeywords(%q) = %d, want %d", tt.line, got, tt.expected)
		}
	}
}

// --- Benchmarks ---

func BenchmarkCountLines_Small(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "small.go")
	content := "package main\n\nimport \"fmt\"\n\n// comment\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CountLines(path, "Go", []string{"//"}, false)
	}
}

func BenchmarkCountTotalLines_Small(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "small.go")
	content := "package main\n\nimport \"fmt\"\n\n// comment\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CountTotalLines(path, "Go")
	}
}

func BenchmarkCountLines_Medium(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "medium.go")
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("// comment line\ncode := doSomething()\n\n")
	}
	os.WriteFile(path, []byte(sb.String()), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CountLines(path, "Go", []string{"//"}, false)
	}
}

func BenchmarkCountTotalLines_Medium(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "medium.go")
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("// comment line\ncode := doSomething()\n\n")
	}
	os.WriteFile(path, []byte(sb.String()), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CountTotalLines(path, "Go")
	}
}

func BenchmarkCountLines_Large(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "large.go")
	var sb strings.Builder
	for i := 0; i < 50000; i++ {
		sb.WriteString("// comment line\ncode := doSomething()\n\n")
	}
	os.WriteFile(path, []byte(sb.String()), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CountLines(path, "Go", []string{"//"}, false)
	}
}

func BenchmarkCountTotalLines_Large(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "large.go")
	var sb strings.Builder
	for i := 0; i < 50000; i++ {
		sb.WriteString("// comment line\ncode := doSomething()\n\n")
	}
	os.WriteFile(path, []byte(sb.String()), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CountTotalLines(path, "Go")
	}
}

func BenchmarkCountLines_WithComplexity(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "complex.go")
	var sb strings.Builder
	for i := 0; i < 10000; i++ {
		sb.WriteString("if x > 0 {\n\tfor i := range items {\n\t\tswitch v {\n\t\tcase 1:\n\t\t}\n\t}\n}\n\n")
	}
	os.WriteFile(path, []byte(sb.String()), 0644)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CountLines(path, "Go", []string{"//"}, true)
	}
}
