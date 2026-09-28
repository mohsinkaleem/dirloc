package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/mohsinkaleem/dirloc/aggregator"
	"github.com/mohsinkaleem/dirloc/output"
	"github.com/mohsinkaleem/dirloc/scanner"
	"github.com/mohsinkaleem/dirloc/types"
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
	cfg            types.ScanConfig
	outputJSON     bool
	outputMD       bool
	noColor        bool
	maxFileSizeStr string
	listLangs      bool
	cpuProfile     string
	memProfile     string
)

func init() {
	f := rootCmd.Flags()
	f.IntVarP(&cfg.TopK, "top-k", "k", 15, "Number of top files/dirs to display")
	f.StringSliceVarP(&cfg.ExcludeDirs, "exclude-dir", "e", nil, "Additional directory names to ignore")
	f.StringSliceVar(&cfg.ExcludeExts, "exclude-ext", nil, "Additional file extensions to ignore")
	f.StringSliceVar(&cfg.ExcludeFiles, "exclude-file", nil, "Additional file names or glob patterns to ignore (e.g. config.json, *_test.go)")
	f.StringSliceVar(&cfg.IncludeExts, "include-ext", nil, "Only include files with these extensions (e.g. go,py)")
	f.StringSliceVar(&cfg.IncludeLangs, "include-lang", nil, "Only include files of these languages, case-insensitive (e.g. go,python)")
	f.BoolVar(&cfg.IncludeDocs, "include-docs", false, "Also count documentation/data files (Markdown, JSON, YAML, ...)")
	f.BoolVar(&cfg.SkipGenerated, "skip-generated", false, "Skip generated files (\"Code generated\", \"DO NOT EDIT\", \"@generated\" headers)")
	f.IntVarP(&cfg.Workers, "workers", "w", runtime.NumCPU(), "Number of parallel worker goroutines")
	f.BoolVarP(&cfg.ShowLang, "lang", "l", false, "Show language breakdown")
	f.BoolVarP(&cfg.ShowComplexity, "complexity", "c", false, "Show complexity column")
	f.StringVarP(&cfg.Format, "format", "o", "table", "Output format: "+strings.Join(output.Formats, ", "))
	f.BoolVar(&outputJSON, "json", false, "Shortcut for --format json")
	f.BoolVar(&outputMD, "md", false, "Shortcut for --format md")
	f.BoolVar(&noColor, "no-color", false, "Disable colours (also honours NO_COLOR)")
	f.BoolVar(&cfg.NoTopFiles, "no-top-files", false, "Suppress top files list")
	f.BoolVar(&cfg.NoTopDirs, "no-top-dirs", false, "Suppress top dirs list")
	f.StringVarP(&cfg.SortBy, "sort", "s", "code", "Sort by: code, total, files")
	f.StringVar(&maxFileSizeStr, "max-file-size", "5MB", "Skip files larger than this (e.g., 10MB, 500KB)")
	f.BoolVar(&cfg.UseGitignore, "gitignore", false, "Respect .gitignore files")
	f.BoolVar(&cfg.UseCache, "cache", false, "Cache results in .dirlocache for faster re-scans")
	f.BoolVar(&cfg.NoProgress, "no-progress", false, "Disable progress indicator")
	f.IntVar(&cfg.MaxDepth, "depth", 0, "Maximum directory depth to scan (0 = unlimited)")
	f.BoolVar(&listLangs, "list-langs", false, "List supported languages and exit")
	f.StringVar(&cpuProfile, "cpuprofile", "", "Write CPU profile to `file` (analyzed with go tool pprof)")
	f.StringVar(&memProfile, "memprofile", "", "Write memory profile to `file` (analyzed with go tool pprof)")

	rootCmd.Version = Version
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// validateFlags resolves shortcut flags and checks flag values.
func validateFlags(cmd *cobra.Command) error {
	if (outputJSON && outputMD) || ((outputJSON || outputMD) && cmd.Flags().Changed("format")) {
		return errors.New("use only one of --format, --json, --md")
	}
	if outputJSON {
		cfg.Format = "json"
	}
	if outputMD {
		cfg.Format = "md"
	}
	if !slices.Contains(output.Formats, cfg.Format) {
		return fmt.Errorf("invalid --format %q: must be one of %s", cfg.Format, strings.Join(output.Formats, ", "))
	}
	switch cfg.SortBy {
	case "code", "total", "files":
	default:
		return fmt.Errorf("invalid --sort value %q: must be code, total, or files", cfg.SortBy)
	}
	if cfg.TopK < 0 {
		return errors.New("--top-k must not be negative")
	}
	if cfg.Workers < 1 {
		return errors.New("--workers must be at least 1")
	}

	maxFileSize, err := parseSize(maxFileSizeStr)
	if err != nil {
		return fmt.Errorf("invalid --max-file-size %q: %w", maxFileSizeStr, err)
	}
	cfg.MaxFileSize = maxFileSize
	return nil
}

func runScan(cmd *cobra.Command, args []string) error {
	if listLangs {
		printLanguages(os.Stdout)
		return nil
	}
	if err := validateFlags(cmd); err != nil {
		return err
	}
	// Past flag validation, errors are runtime failures; don't dump usage for them.
	cmd.SilenceUsage = true

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

	cfg.RootPath = "."
	if len(args) > 0 {
		cfg.RootPath = args[0]
	}
	root := cfg.RootPath

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	start := time.Now()

	ignore := scanner.NewIgnoreRules(cfg.ExcludeDirs, cfg.ExcludeExts, cfg.ExcludeFiles)
	ignore.SetIncludes(cfg.IncludeExts, cfg.IncludeLangs, cfg.IncludeDocs)

	var gitMatcher *scanner.GitIgnoreMatcher
	if cfg.UseGitignore {
		gitMatcher = scanner.NewGitIgnoreMatcher()
	}

	var progress *scanner.Progress
	if !cfg.NoProgress {
		progress = scanner.NewProgress() // returns nil if stderr is not a TTY
	}

	var cache *scanner.Cache
	if cfg.UseCache {
		cache = scanner.LoadCache(root)
	}

	progress.Start()
	paths, warnings, err := scanner.Walk(ctx, root, ignore, cfg.MaxFileSize, gitMatcher, progress, cfg.MaxDepth)
	if err != nil {
		progress.Stop()
		return err
	}

	// Buffer warnings so they don't interleave with the progress line.
	var warns []string
	warnsDone := make(chan struct{})
	go func() {
		defer close(warnsDone)
		for w := range warnings {
			warns = append(warns, w)
		}
	}()

	allResults := make([]types.FileResult, 0, 256)
	for r := range scanner.ProcessFiles(ctx, paths, cfg, cache) {
		allResults = append(allResults, r)
	}

	progress.Stop()
	<-warnsDone
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, w)
	}

	if ctx.Err() != nil {
		return errors.New("interrupted")
	}

	if cache != nil {
		if err := cache.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot write cache: %v\n", err)
		}
	}

	elapsed := time.Since(start)

	if len(allResults) == 0 && cfg.Format == "table" {
		fmt.Println("No code files found.")
		return nil
	}

	dirStats := aggregator.AggregateDirs(allResults)
	langSummaries := aggregator.AggregateLangs(allResults)

	return output.Render(os.Stdout, output.Report{
		Summary: aggregator.SummaryTotals(allResults, dirStats, len(langSummaries)),
		Files:   aggregator.TopKFiles(allResults, cfg.TopK, cfg.SortBy),
		Dirs:    aggregator.TopKDirs(dirStats, cfg.TopK, cfg.SortBy),
		Langs:   langSummaries,
		Config:  cfg,
		Elapsed: elapsed,
		Color:   output.ColorEnabled(os.Stdout, noColor),
	})
}

func printLanguages(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "LANGUAGE\tKIND\tMATCHES")
	for _, l := range scanner.Languages() {
		kind := "code"
		if l.Doc {
			kind = "docs"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", l.Name, kind, strings.Join(l.Patterns, " "))
	}
	tw.Flush()
	fmt.Fprintln(w, "\ndocs languages are skipped unless --include-docs or --include-lang/--include-ext selects them.")
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
