package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dirloc/dirloc/aggregator"
	"github.com/dirloc/dirloc/output"
	"github.com/dirloc/dirloc/scanner"
	"github.com/dirloc/dirloc/types"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "dirloc [path]",
	Short: "Fast directory code scanner & summarizer",
	Long:  "dirloc recursively scans a directory tree, counts lines of code per file,\naggregates stats by directory and language, and reports Top-K files/directories.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runScan,
}

var (
	topK           int
	excludeDirs    []string
	excludeExts    []string
	excludeFile    string
	workers        int
	showLang       bool
	showComplexity bool
	outputJSON     bool
	outputMD       bool
	noTopFiles     bool
	noTopDirs      bool
	sortBy         string
	maxFileSizeStr string
	cpuProfile     string
	memProfile     string
)

func init() {
	rootCmd.Flags().IntVarP(&topK, "top-k", "k", 10, "Number of top files/dirs to display")
	rootCmd.Flags().StringSliceVarP(&excludeDirs, "exclude-dir", "e", nil, "Additional directory names to ignore")
	rootCmd.Flags().StringSliceVar(&excludeExts, "exclude-ext", nil, "Additional file extensions to ignore")
	rootCmd.Flags().StringVar(&excludeFile, "exclude-file", "", "Path to exclude file (default: .dirlocignore in scanned dir)")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "Number of parallel worker goroutines")
	rootCmd.Flags().BoolVarP(&showLang, "lang", "l", false, "Show language breakdown")
	rootCmd.Flags().BoolVarP(&showComplexity, "complexity", "c", false, "Show complexity column")
	rootCmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")
	rootCmd.Flags().BoolVar(&outputMD, "md", false, "Output as Markdown")
	rootCmd.Flags().BoolVar(&noTopFiles, "no-top-files", false, "Suppress top files list")
	rootCmd.Flags().BoolVar(&noTopDirs, "no-top-dirs", false, "Suppress top dirs list")
	rootCmd.Flags().StringVarP(&sortBy, "sort", "s", "code", "Sort by: code, total, files")
	rootCmd.Flags().StringVar(&maxFileSizeStr, "max-file-size", "10MB", "Skip files larger than this (e.g., 10MB, 500KB)")
	rootCmd.Flags().StringVar(&cpuProfile, "cpuprofile", "", "Write CPU profile to `file` (analyzed with go tool pprof)")
	rootCmd.Flags().StringVar(&memProfile, "memprofile", "", "Write memory profile to `file` (analyzed with go tool pprof)")

	rootCmd.Version = Version
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runScan(cmd *cobra.Command, args []string) error {
	// Validate flags
	if outputJSON && outputMD {
		return fmt.Errorf("cannot use --json and --md together")
	}
	switch sortBy {
	case "code", "total", "files":
	default:
		return fmt.Errorf("invalid --sort value %q: must be code, total, or files", sortBy)
	}

	maxFileSize, err := parseSize(maxFileSizeStr)
	if err != nil {
		return fmt.Errorf("invalid --max-file-size %q: %w", maxFileSizeStr, err)
	}

	// Start CPU profiling before any scan work.
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return fmt.Errorf("could not create CPU profile %q: %w", cpuProfile, err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	// Heap profile is written after the scan completes (LIFO defer order ensures
	// this runs before StopCPUProfile so both profiles capture the full run).
	if memProfile != "" {
		defer func() {
			f, err := os.Create(memProfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not create memory profile %q: %v\n", memProfile, err)
				return
			}
			defer f.Close()
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "could not write memory profile: %v\n", err)
			}
		}()
	}

	root := "."
	if len(args) > 0 {
		root = args[0]
	}

	config := types.ScanConfig{
		RootPath:       root,
		ExcludeDirs:    excludeDirs,
		ExcludeExts:    excludeExts,
		Workers:        workers,
		TopK:           topK,
		ShowLang:       showLang,
		ShowComplexity: showComplexity,
		OutputJSON:     outputJSON,
		OutputMD:       outputMD,
		NoTopFiles:     noTopFiles,
		NoTopDirs:      noTopDirs,
		SortBy:         sortBy,
		MaxFileSize:    maxFileSize,
	}

	// Setup context with signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	start := time.Now()

	// Build ignore rules
	ignore := scanner.NewIgnoreRules(config.ExcludeDirs, config.ExcludeExts)

	// Load .dirlocignore file
	ignoreFile := excludeFile
	if ignoreFile == "" {
		ignoreFile = filepath.Join(root, ".dirlocignore")
	}
	if _, err := os.Stat(ignoreFile); err == nil {
		if err := ignore.LoadIgnoreFile(ignoreFile); err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot read %s: %v\n", ignoreFile, err)
		}
	}

	// Walk the directory tree
	paths, warnings, err := scanner.Walk(ctx, root, ignore, config.MaxFileSize)
	if err != nil {
		return err
	}

	// Drain warnings in background
	go func() {
		for w := range warnings {
			fmt.Fprintln(os.Stderr, w)
		}
	}()

	// Process files with worker pool
	results := scanner.ProcessFiles(ctx, paths, config)

	// Collect all results
	var allResults []types.FileResult
	for r := range results {
		allResults = append(allResults, r)
	}

	elapsed := time.Since(start)

	if len(allResults) == 0 {
		fmt.Println("No code files found.")
		return nil
	}

	// Aggregate
	dirStats := aggregator.AggregateDirs(allResults)
	langSummaries := aggregator.AggregateLangs(allResults)
	topFiles := aggregator.TopKFiles(allResults, config.TopK, config.SortBy)
	topDirs := aggregator.TopKDirs(dirStats, config.TopK, config.SortBy)
	summary := aggregator.SummaryTotals(allResults, dirStats, len(langSummaries))

	// Output
	switch {
	case config.OutputJSON:
		return output.RenderJSON(summary, topFiles, topDirs, langSummaries, config, elapsed)
	case config.OutputMD:
		output.RenderMarkdown(summary, topFiles, topDirs, langSummaries, config, elapsed)
	default:
		output.RenderTable(summary, topFiles, topDirs, langSummaries, config, elapsed)
	}

	return nil
}

// parseSize parses a human-readable size string like "10MB" into bytes.
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" || s == "0" {
		return 0, nil
	}

	// Check longest suffixes first to avoid "B" matching before "MB"
	type suffixMult struct {
		suffix string
		mult   int64
	}
	multipliers := []suffixMult{
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, sm := range multipliers {
		if strings.HasSuffix(s, sm.suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(s, sm.suffix))
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("cannot parse number: %w", err)
			}
			return int64(n * float64(sm.mult)), nil
		}
	}

	// Try plain number (bytes)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse size: %w", err)
	}
	return n, nil
}
