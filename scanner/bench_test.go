package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dirloc/dirloc/aggregator"
	"github.com/dirloc/dirloc/types"
)

// --- helpers ----------------------------------------------------------------

// fileSpec describes one synthetic source file to generate.
type fileSpec struct {
	ext     string
	content func(pkg, idx int) string
}

var fileSpecs = []fileSpec{
	{
		ext: ".go",
		content: func(pkg, idx int) string {
			return fmt.Sprintf(`package pkg%d

import "fmt"

// Func%d demonstrates a typical Go function with branching logic.
func Func%d(x int) int {
	if x > 0 {
		for i := 0; i < x; i++ {
			fmt.Println(i)
		}
	} else if x < 0 {
		switch x {
		case -1:
			return -1
		default:
			return x * 2
		}
	}
	return x
}
`, pkg, idx, idx)
		},
	},
	{
		ext: ".py",
		content: func(pkg, idx int) string {
			return fmt.Sprintf(`# Module pkg%d, function %d
def func_%d(x):
    """Return x after conditional processing."""
    if x > 0:
        for i in range(x):
            print(i)
    elif x < 0:
        return x * 2
    return x
`, pkg, idx, idx)
		},
	},
	{
		ext: ".ts",
		content: func(pkg, idx int) string {
			return fmt.Sprintf(`// pkg%d / func%d
export function func%d(x: number): number {
    // conditional branching
    if (x > 0) {
        for (let i = 0; i < x; i++) {
            console.log(i);
        }
    } else if (x < 0) {
        switch (x) {
            case -1: return -1;
            default: return x * 2;
        }
    }
    return x;
}
`, pkg, idx, idx)
		},
	},
	{
		ext: ".java",
		content: func(pkg, idx int) string {
			return fmt.Sprintf(`// pkg%d
public class Class%d {
    /** Process x with branching logic. */
    public int method%d(int x) {
        if (x > 0) {
            for (int i = 0; i < x; i++) {
                System.out.println(i);
            }
        } else if (x < 0) {
            return x * 2;
        }
        return x;
    }
}
`, pkg, idx, idx)
		},
	},
	{
		ext: ".rs",
		content: func(pkg, idx int) string {
			return fmt.Sprintf(`// pkg%d func%d
pub fn func_%d(x: i32) -> i32 {
    if x > 0 {
        for i in 0..x {
            println!("{}", i);
        }
    } else if x < 0 {
        match x {
            -1 => return -1,
            _ => return x * 2,
        }
    }
    x
}
`, pkg, idx, idx)
		},
	},
}

// generateRepo creates numDirs directories each containing filesPerDir files
// with rotating languages. Returns the total file count created.
func generateRepo(t testing.TB, root string, numDirs, filesPerDir int) int {
	t.Helper()
	count := 0
	for i := 0; i < numDirs; i++ {
		sub := filepath.Join(root, fmt.Sprintf("pkg%d", i))
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
		for j := 0; j < filesPerDir; j++ {
			spec := fileSpecs[(i+j)%len(fileSpecs)]
			p := filepath.Join(sub, fmt.Sprintf("file%d%s", j, spec.ext))
			if err := os.WriteFile(p, []byte(spec.content(i, j)), 0644); err != nil {
				t.Fatalf("write %s: %v", p, err)
			}
			count++
		}
	}
	return count
}

// runFullPipeline executes Walk → ProcessFiles and returns all FileResults.
func runFullPipeline(t testing.TB, root string, workers int, showLang bool) []types.FileResult {
	t.Helper()
	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), root, ignore, 10*1024*1024)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	go func() {
		for range warnings {
		}
	}()

	cfg := types.ScanConfig{
		RootPath: root,
		Workers:  workers,
		ShowLang: showLang,
	}
	results := ProcessFiles(context.Background(), paths, cfg)
	var all []types.FileResult
	for r := range results {
		all = append(all, r)
	}
	return all
}

// --- Stress Tests -----------------------------------------------------------

// TestStress_FullPipeline_10K validates the complete scan+aggregate pipeline
// against a 10 000-file synthetic repository with five languages.
func TestStress_FullPipeline_10K(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	dir := t.TempDir()
	want := generateRepo(t, dir, 100, 100) // 100 dirs × 100 files = 10 000

	all := runFullPipeline(t, dir, runtime.NumCPU(), true)

	if len(all) != want {
		t.Errorf("file count: got %d, want %d", len(all), want)
	}

	for _, r := range all {
		if r.Error != "" {
			t.Errorf("%s: unexpected error: %s", r.Path, r.Error)
		}
		if r.Total == 0 {
			t.Errorf("%s: total lines == 0", r.Path)
		}
		if r.Code == 0 {
			t.Errorf("%s: code lines == 0 (detailed scan should find code)", r.Path)
		}
	}

	// Verify aggregation correctness.
	dirStats := aggregator.AggregateDirs(all)
	if len(dirStats) == 0 {
		t.Error("AggregateDirs: empty result")
	}

	langSummaries := aggregator.AggregateLangs(all)
	if len(langSummaries) < len(fileSpecs) {
		t.Errorf("AggregateLangs: got %d languages, want at least %d", len(langSummaries), len(fileSpecs))
	}

	summary := aggregator.SummaryTotals(all, dirStats, len(langSummaries))
	if summary.TotalFiles != want {
		t.Errorf("SummaryTotals.TotalFiles: got %d, want %d", summary.TotalFiles, want)
	}
	if summary.TotalCode == 0 {
		t.Error("SummaryTotals.TotalCode == 0")
	}

	t.Logf("10K stress: %d files, %d dirs, %d languages, %d code lines, %d total lines",
		summary.TotalFiles, len(dirStats), len(langSummaries), summary.TotalCode, summary.TotalLines)
}

// TestStress_FullPipeline_WithComplexity verifies complexity counting at scale.
func TestStress_FullPipeline_WithComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	dir := t.TempDir()
	want := generateRepo(t, dir, 50, 40) // 2 000 files

	ignore := NewIgnoreRules(nil, nil)
	paths, warnings, err := Walk(context.Background(), dir, ignore, 10*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for range warnings {
		}
	}()

	cfg := types.ScanConfig{
		RootPath:       dir,
		Workers:        runtime.NumCPU(),
		ShowLang:       true,
		ShowComplexity: true,
	}
	results := ProcessFiles(context.Background(), paths, cfg)
	var all []types.FileResult
	for r := range results {
		all = append(all, r)
	}

	if len(all) != want {
		t.Errorf("file count: got %d, want %d", len(all), want)
	}

	complexFiles := 0
	for _, r := range all {
		if r.Complexity > 0 {
			complexFiles++
		}
	}
	if complexFiles == 0 {
		t.Error("expected at least some files with complexity > 0")
	}
	t.Logf("complexity stress: %d/%d files reported complexity > 0", complexFiles, want)
}

// --- Benchmarks -------------------------------------------------------------

// BenchmarkFullPipeline_10KFiles measures end-to-end throughput (Walk +
// ProcessFiles) against a pre-generated 10 000-file repo.
func BenchmarkFullPipeline_10KFiles(b *testing.B) {
	dir := b.TempDir()
	fileCount := generateRepo(b, dir, 100, 100)
	ignore := NewIgnoreRules(nil, nil)
	cfg := types.ScanConfig{
		RootPath: dir,
		Workers:  runtime.NumCPU(),
		ShowLang: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		paths, warnings, _ := Walk(context.Background(), dir, ignore, 10*1024*1024)
		go func() {
			for range warnings {
			}
		}()
		results := ProcessFiles(context.Background(), paths, cfg)
		for range results {
		}
	}

	b.ReportMetric(float64(fileCount), "files/op")
}

// BenchmarkFullPipeline_FastPath benchmarks the fast (total-lines-only) path.
func BenchmarkFullPipeline_FastPath(b *testing.B) {
	dir := b.TempDir()
	fileCount := generateRepo(b, dir, 100, 100)
	ignore := NewIgnoreRules(nil, nil)
	cfg := types.ScanConfig{
		RootPath: dir,
		Workers:  runtime.NumCPU(),
		ShowLang: false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		paths, warnings, _ := Walk(context.Background(), dir, ignore, 10*1024*1024)
		go func() {
			for range warnings {
			}
		}()
		results := ProcessFiles(context.Background(), paths, cfg)
		for range results {
		}
	}

	b.ReportMetric(float64(fileCount), "files/op")
}

// BenchmarkWorkerScaling measures how throughput scales with worker count.
func BenchmarkWorkerScaling(b *testing.B) {
	dir := b.TempDir()
	var filePaths []string
	for i := 0; i < 5000; i++ {
		sub := filepath.Join(dir, fmt.Sprintf("pkg%d", i/50))
		os.MkdirAll(sub, 0755)
		spec := fileSpecs[i%len(fileSpecs)]
		p := filepath.Join(sub, fmt.Sprintf("file%d%s", i, spec.ext))
		os.WriteFile(p, []byte(spec.content(i/50, i)), 0644)
		filePaths = append(filePaths, p)
	}

	for _, w := range []int{1, 2, 4, 8, runtime.NumCPU()} {
		w := w
		b.Run(fmt.Sprintf("workers=%d", w), func(b *testing.B) {
			cfg := types.ScanConfig{
				RootPath: dir,
				Workers:  w,
				ShowLang: true,
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ch := make(chan string, len(filePaths))
				for _, p := range filePaths {
					ch <- p
				}
				close(ch)
				results := ProcessFiles(context.Background(), ch, cfg)
				for range results {
				}
			}
		})
	}
}

// BenchmarkAggregation measures the aggregation layer on a large result set.
func BenchmarkAggregation(b *testing.B) {
	dir := b.TempDir()
	all := runFullPipeline(b, dir, runtime.NumCPU(), true)
	// Pre-generate once; just benchmark aggregation math.
	_ = generateRepo(b, dir, 100, 100)
	all = runFullPipeline(b, dir, runtime.NumCPU(), true)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dirStats := aggregator.AggregateDirs(all)
		_ = aggregator.AggregateLangs(all)
		_ = aggregator.TopKFiles(all, 10, "code")
		_ = aggregator.TopKDirs(dirStats, 10, "code")
		_ = aggregator.SummaryTotals(all, dirStats, 5)
	}
}
