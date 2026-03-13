package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dirloc/dirloc/internal/analyzer"
	"github.com/dirloc/dirloc/pkg/types"
)

func TestWalk_BasicDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create some code files
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "helper.go"), []byte("package sub\n"), 0644)

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), dir, ignore, 10*1024*1024)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	var files []string
	for p := range paths {
		files = append(files, p)
	}
	for range warnings {
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

func TestWalk_SkipsIgnoredDirs(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0755)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg.js"), []byte("var x;\n"), 0644)

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), dir, ignore, 10*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for p := range paths {
		files = append(files, p)
	}
	for range warnings {
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file (node_modules should be skipped), got %d", len(files))
	}
}

func TestWalk_SkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)

	// Create a binary file with a known extension
	binContent := make([]byte, 100)
	binContent[50] = 0 // null byte
	os.WriteFile(filepath.Join(dir, "data.go"), binContent, 0644)

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), dir, ignore, 10*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for p := range paths {
		files = append(files, p)
	}
	for range warnings {
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file (binary should be skipped), got %d", len(files))
	}
}

func TestWalk_MaxFileSize(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "small.go"), []byte("package main\n"), 0644)
	// Create a file larger than 100 bytes
	os.WriteFile(filepath.Join(dir, "large.go"), make([]byte, 200), 0644)

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), dir, ignore, 100)
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for p := range paths {
		files = append(files, p)
	}
	// Drain warnings
	for range warnings {
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file (large file should be skipped), got %d", len(files))
	}
}

func TestWalk_CancelContext(t *testing.T) {
	dir := t.TempDir()

	// Create many files
	for i := 0; i < 100; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.go", i)), []byte("package main\n"), 0644)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(ctx, dir, ignore, 10*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for range paths {
		count++
	}
	for range warnings {
	}

	// With cancelled context, should get fewer than 100 files
	if count >= 100 {
		t.Errorf("expected fewer files with cancelled context, got %d", count)
	}
}

func TestWalk_NonExistentDir(t *testing.T) {
	ignore := NewIgnoreRules(nil, nil)
	_, _, err := Walk(context.Background(), "/nonexistent/dir", ignore, 10*1024*1024)
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestProcessFiles_Basic(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "main.go")
	f2 := filepath.Join(dir, "util.go")
	os.WriteFile(f1, []byte("package main\n\nfunc main() {}\n"), 0644)
	os.WriteFile(f2, []byte("package main\n\n// util\nfunc util() {}\n"), 0644)

	paths := make(chan string, 2)
	paths <- f1
	paths <- f2
	close(paths)

	config := types.ScanConfig{
		RootPath:       dir,
		Workers:        2,
		ShowLang:       true,
		ShowComplexity: false,
	}

	results := ProcessFiles(context.Background(), paths, config)

	var all []types.FileResult
	for r := range results {
		all = append(all, r)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 results, got %d", len(all))
	}

	for _, r := range all {
		if r.Language != "Go" {
			t.Errorf("expected Go language, got %q", r.Language)
		}
		if r.Total == 0 {
			t.Errorf("expected non-zero total for %s", r.Path)
		}
	}
}

func TestProcessFiles_FastPath(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "main.go")
	os.WriteFile(f1, []byte("package main\n\n// comment\nfunc main() {}\n"), 0644)

	paths := make(chan string, 1)
	paths <- f1
	close(paths)

	config := types.ScanConfig{
		RootPath: dir,
		Workers:  1,
		ShowLang: false, // fast path
	}

	results := ProcessFiles(context.Background(), paths, config)

	var all []types.FileResult
	for r := range results {
		all = append(all, r)
	}

	if len(all) != 1 {
		t.Fatalf("expected 1 result, got %d", len(all))
	}

	r := all[0]
	if r.Total != 4 {
		t.Errorf("total = %d, want 4", r.Total)
	}
	// Fast path: code/comment/blank should be 0
	if r.Code != 0 || r.Comment != 0 || r.Blank != 0 {
		t.Errorf("fast path should not compute code/comment/blank, got code=%d comment=%d blank=%d",
			r.Code, r.Comment, r.Blank)
	}
}

// --- Stress Tests ---

func TestStress_LargeDirectoryTree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	dir := t.TempDir()

	// Create a directory tree with 1000 files across 50 directories
	fileCount := 0
	for i := 0; i < 50; i++ {
		subdir := filepath.Join(dir, fmt.Sprintf("pkg%d", i))
		os.MkdirAll(subdir, 0755)
		for j := 0; j < 20; j++ {
			path := filepath.Join(subdir, fmt.Sprintf("file%d.go", j))
			content := fmt.Sprintf("package pkg%d\n\n// file %d\nfunc Func%d() {\n\treturn\n}\n", i, j, j)
			os.WriteFile(path, []byte(content), 0644)
			fileCount++
		}
	}

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), dir, ignore, 10*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for range warnings {
		}
	}()

	config := types.ScanConfig{
		RootPath: dir,
		Workers:  4,
		ShowLang: false,
	}

	results := ProcessFiles(context.Background(), paths, config)

	var all []types.FileResult
	for r := range results {
		all = append(all, r)
	}

	if len(all) != fileCount {
		t.Errorf("expected %d results, got %d", fileCount, len(all))
	}

	// Verify all results have non-zero total
	for _, r := range all {
		if r.Total == 0 {
			t.Errorf("file %s has 0 total lines", r.Path)
		}
		if r.Error != "" {
			t.Errorf("file %s has error: %s", r.Path, r.Error)
		}
	}
}

func TestStress_DeeplyNestedDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	dir := t.TempDir()

	// Create a deeply nested directory (20 levels)
	current := dir
	for i := 0; i < 20; i++ {
		current = filepath.Join(current, fmt.Sprintf("level%d", i))
		os.MkdirAll(current, 0755)
		os.WriteFile(filepath.Join(current, "file.go"), []byte("package main\n"), 0644)
	}

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), dir, ignore, 10*1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for p := range paths {
		files = append(files, p)
	}
	for range warnings {
	}

	if len(files) != 20 {
		t.Errorf("expected 20 files in nested dirs, got %d", len(files))
	}
}

// --- Benchmarks ---

func BenchmarkWalk_1KFiles(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 1000; i++ {
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.go", i)), []byte("package main\n"), 0644)
	}

	ignore := NewIgnoreRules(nil, nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		paths, warnings, _ := Walk(context.Background(), dir, ignore, 10*1024*1024)
		for range paths {
		}
		for range warnings {
		}
	}
}

func BenchmarkProcessFiles_1KFiles(b *testing.B) {
	dir := b.TempDir()
	var filePaths []string
	for i := 0; i < 1000; i++ {
		p := filepath.Join(dir, fmt.Sprintf("file%d.go", i))
		os.WriteFile(p, []byte("package main\n\n// comment\nfunc main() {}\n"), 0644)
		filePaths = append(filePaths, p)
	}

	config := types.ScanConfig{
		RootPath: dir,
		Workers:  4,
		ShowLang: false,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		paths := make(chan string, len(filePaths))
		for _, p := range filePaths {
			paths <- p
		}
		close(paths)

		results := ProcessFiles(context.Background(), paths, config)
		for range results {
		}
	}
}

func BenchmarkProcessFiles_1KFiles_Detailed(b *testing.B) {
	dir := b.TempDir()
	var filePaths []string
	for i := 0; i < 1000; i++ {
		p := filepath.Join(dir, fmt.Sprintf("file%d.go", i))
		os.WriteFile(p, []byte("package main\n\n// comment\nfunc main() {}\n"), 0644)
		filePaths = append(filePaths, p)
	}

	config := types.ScanConfig{
		RootPath: dir,
		Workers:  4,
		ShowLang: true,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		paths := make(chan string, len(filePaths))
		for _, p := range filePaths {
			paths <- p
		}
		close(paths)

		results := ProcessFiles(context.Background(), paths, config)
		for range results {
		}
	}
}

// init loads the language database for tests
func init() {
	// Load the language database
	data, err := os.ReadFile("../../languages.json")
	if err != nil {
		// Try relative path from test
		data, err = os.ReadFile("../../languages.json")
		if err != nil {
			panic("cannot load languages.json for tests: " + err.Error())
		}
	}
	analyzer.InitLanguages(data)
}
