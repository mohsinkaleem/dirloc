package aggregator

import (
	"path/filepath"
	"testing"

	"github.com/dirloc/dirloc/types"
)

func makeResults(n int) []types.FileResult {
	results := make([]types.FileResult, n)
	dirs := []string{"src", "src/pkg", "src/pkg/utils", "lib", "lib/core"}
	langs := []string{"Go", "Python", "JavaScript", "Rust"}
	for i := range results {
		dir := dirs[i%len(dirs)]
		results[i] = types.FileResult{
			Path:     dir + "/file" + string(rune('a'+i%26)) + ".go",
			Language: langs[i%len(langs)],
			Code:     100 + i%50,
			Comment:  20 + i%10,
			Blank:    10 + i%5,
			Total:    130 + i%65,
		}
	}
	return results
}

// --- AggregateDirs tests ---

func TestAggregateDirs_Simple(t *testing.T) {
	results := []types.FileResult{
		{Path: "src/main.go", Language: "Go", Code: 100, Comment: 20, Blank: 10, Total: 130},
		{Path: "src/util.go", Language: "Go", Code: 50, Comment: 10, Blank: 5, Total: 65},
	}

	dirs := AggregateDirs(results)

	// "src" directory should have aggregated stats
	srcDir, ok := dirs["src"]
	if !ok {
		t.Fatal("expected 'src' directory in results")
	}
	if srcDir.Files != 2 {
		t.Errorf("src files = %d, want 2", srcDir.Files)
	}
	if srcDir.Code != 150 {
		t.Errorf("src code = %d, want 150", srcDir.Code)
	}
	if srcDir.Total != 195 {
		t.Errorf("src total = %d, want 195", srcDir.Total)
	}
}

func TestAggregateDirs_HierarchyRollup(t *testing.T) {
	results := []types.FileResult{
		{Path: filepath.Join("src", "pkg", "file.go"), Language: "Go", Code: 100, Total: 100},
	}

	dirs := AggregateDirs(results)

	srcPkg := filepath.Join("src", "pkg")
	// Both "src/pkg" and "src" should have the file's stats
	if _, ok := dirs[srcPkg]; !ok {
		t.Fatalf("expected %q in results, got keys: %v", srcPkg, mapKeys(dirs))
	}
	if _, ok := dirs["src"]; !ok {
		t.Fatal("expected 'src' in results (hierarchy rollup)")
	}
	if dirs["src"].Code != 100 {
		t.Errorf("src code = %d, want 100", dirs["src"].Code)
	}
}

func mapKeys(m map[string]*types.DirStats) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestAggregateDirs_SkipsErrors(t *testing.T) {
	results := []types.FileResult{
		{Path: "src/main.go", Language: "Go", Code: 100, Total: 100},
		{Path: "src/bad.go", Language: "Go", Error: "read error"},
	}

	dirs := AggregateDirs(results)
	srcDir := dirs["src"]
	if srcDir.Files != 1 {
		t.Errorf("src files = %d, want 1 (should skip errors)", srcDir.Files)
	}
}

// --- AggregateLangs tests ---

func TestAggregateLangs_Simple(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Language: "Go", Code: 100, Total: 130},
		{Path: "b.go", Language: "Go", Code: 200, Total: 250},
		{Path: "c.py", Language: "Python", Code: 50, Total: 60},
	}

	langs := AggregateLangs(results)

	if len(langs) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(langs))
	}
	// Should be sorted by code descending → Go first
	if langs[0].Language != "Go" {
		t.Errorf("first language = %q, want Go", langs[0].Language)
	}
	if langs[0].Code != 300 {
		t.Errorf("Go code = %d, want 300", langs[0].Code)
	}
	if langs[0].Files != 2 {
		t.Errorf("Go files = %d, want 2", langs[0].Files)
	}
}

func TestAggregateLangs_SkipsErrors(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Language: "Go", Code: 100, Total: 100},
		{Path: "b.go", Language: "Go", Code: 0, Error: "fail"},
	}

	langs := AggregateLangs(results)
	if len(langs) != 1 {
		t.Fatalf("expected 1 language, got %d", len(langs))
	}
	if langs[0].Files != 1 {
		t.Errorf("files = %d, want 1", langs[0].Files)
	}
}

// --- TopKFiles tests ---

func TestTopKFiles_Basic(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Code: 100, Total: 150},
		{Path: "b.go", Code: 200, Total: 250},
		{Path: "c.go", Code: 50, Total: 80},
	}

	top := TopKFiles(results, 2, "code")
	if len(top) != 2 {
		t.Fatalf("expected 2 results, got %d", len(top))
	}
	if top[0].Path != "b.go" {
		t.Errorf("first = %q, want b.go", top[0].Path)
	}
	if top[1].Path != "a.go" {
		t.Errorf("second = %q, want a.go", top[1].Path)
	}
}

func TestTopKFiles_SortByTotal(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Code: 200, Total: 150},
		{Path: "b.go", Code: 100, Total: 250},
	}

	top := TopKFiles(results, 2, "total")
	if top[0].Path != "b.go" {
		t.Errorf("first by total = %q, want b.go", top[0].Path)
	}
}

func TestTopKFiles_KLargerThanResults(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Code: 100, Total: 100},
	}

	top := TopKFiles(results, 10, "code")
	if len(top) != 1 {
		t.Errorf("expected 1 result, got %d", len(top))
	}
}

func TestTopKFiles_FiltersErrors(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Code: 100, Total: 100},
		{Path: "b.go", Code: 200, Total: 200, Error: "failed"},
	}

	top := TopKFiles(results, 10, "code")
	if len(top) != 1 {
		t.Errorf("expected 1 result (error filtered), got %d", len(top))
	}
}

func TestTopKFiles_SortByFilesUsesTotal(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Code: 0, Total: 100},
		{Path: "b.go", Code: 0, Total: 200},
	}

	top := TopKFiles(results, 2, "files")
	if top[0].Path != "b.go" {
		t.Errorf("files sort: first = %q, want b.go (highest total)", top[0].Path)
	}
}

// --- TopKDirs tests ---

func TestTopKDirs_Basic(t *testing.T) {
	dirStats := map[string]*types.DirStats{
		"src":  {Path: "src", Files: 10, Code: 1000, Total: 1500},
		"lib":  {Path: "lib", Files: 5, Code: 500, Total: 700},
		"test": {Path: "test", Files: 3, Code: 200, Total: 300},
	}

	top := TopKDirs(dirStats, 2, "code")
	if len(top) != 2 {
		t.Fatalf("expected 2 dirs, got %d", len(top))
	}
	if top[0].Path != "src" {
		t.Errorf("first dir = %q, want src", top[0].Path)
	}
}

func TestTopKDirs_SortByFiles(t *testing.T) {
	dirStats := map[string]*types.DirStats{
		"src":  {Path: "src", Files: 5, Code: 1000, Total: 1500},
		"lib":  {Path: "lib", Files: 10, Code: 500, Total: 700},
	}

	top := TopKDirs(dirStats, 2, "files")
	if top[0].Path != "lib" {
		t.Errorf("first by files = %q, want lib", top[0].Path)
	}
}

// --- SummaryTotals tests ---

func TestSummaryTotals(t *testing.T) {
	results := []types.FileResult{
		{Path: "a.go", Language: "Go", Code: 100, Comment: 20, Blank: 10, Total: 130},
		{Path: "b.py", Language: "Python", Code: 50, Comment: 5, Blank: 5, Total: 60},
		{Path: "c.go", Language: "Go", Error: "fail"},
	}

	dirStats := map[string]*types.DirStats{
		".": {Path: "."},
	}

	summary := SummaryTotals(results, dirStats, 2)

	if summary.TotalFiles != 2 {
		t.Errorf("total files = %d, want 2", summary.TotalFiles)
	}
	if summary.TotalCode != 150 {
		t.Errorf("total code = %d, want 150", summary.TotalCode)
	}
	if summary.TotalLines != 190 {
		t.Errorf("total lines = %d, want 190", summary.TotalLines)
	}
	if summary.Errors != 1 {
		t.Errorf("errors = %d, want 1", summary.Errors)
	}
	if summary.Languages != 2 {
		t.Errorf("languages = %d, want 2", summary.Languages)
	}
}

// --- Sort stability tests ---

func TestTopKFiles_StableSortByCode(t *testing.T) {
	results := []types.FileResult{
		{Path: "b.go", Code: 100, Total: 200},
		{Path: "a.go", Code: 100, Total: 200},
		{Path: "c.go", Code: 100, Total: 200},
	}

	top := TopKFiles(results, 3, "code")
	// With same Code and Total, should sort by Path ascending
	if top[0].Path != "a.go" || top[1].Path != "b.go" || top[2].Path != "c.go" {
		t.Errorf("unstable sort: got %v, %v, %v", top[0].Path, top[1].Path, top[2].Path)
	}
}

func TestTopKDirs_StableSortByCode(t *testing.T) {
	dirStats := map[string]*types.DirStats{
		"b": {Path: "b", Code: 100, Total: 200},
		"a": {Path: "a", Code: 100, Total: 200},
		"c": {Path: "c", Code: 100, Total: 200},
	}

	top := TopKDirs(dirStats, 3, "code")
	if top[0].Path != "a" || top[1].Path != "b" || top[2].Path != "c" {
		t.Errorf("unstable sort: got %v, %v, %v", top[0].Path, top[1].Path, top[2].Path)
	}
}

// --- Benchmarks ---

func BenchmarkAggregateDirs_1K(b *testing.B) {
	results := makeResults(1000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		AggregateDirs(results)
	}
}

func BenchmarkAggregateDirs_10K(b *testing.B) {
	results := makeResults(10_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		AggregateDirs(results)
	}
}

func BenchmarkAggregateDirs_100K(b *testing.B) {
	results := makeResults(100_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		AggregateDirs(results)
	}
}

func BenchmarkAggregateLangs_10K(b *testing.B) {
	results := makeResults(10_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		AggregateLangs(results)
	}
}

func BenchmarkTopKFiles_10K(b *testing.B) {
	results := makeResults(10_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		TopKFiles(results, 10, "code")
	}
}

func BenchmarkTopKFiles_100K(b *testing.B) {
	results := makeResults(100_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		TopKFiles(results, 10, "code")
	}
}

func BenchmarkSummaryTotals_10K(b *testing.B) {
	results := makeResults(10_000)
	dirs := AggregateDirs(results)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		SummaryTotals(results, dirs, 4)
	}
}
