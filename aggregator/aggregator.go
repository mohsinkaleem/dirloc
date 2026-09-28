package aggregator

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mohsinkaleem/dirloc/types"
)

// AggregateDirs groups file results by directory and rolls up stats.
func AggregateDirs(results []types.FileResult) map[string]*types.DirStats {
	dirs := make(map[string]*types.DirStats)

	for _, r := range results {
		if r.Error != "" {
			continue
		}
		// Walk up the directory hierarchy and add stats to each ancestor
		for d := filepath.Dir(r.Path); ; d = filepath.Dir(d) {
			ds, ok := dirs[d]
			if !ok {
				ds = &types.DirStats{Path: d}
				dirs[d] = ds
			}
			ds.Files++
			ds.Code += r.Code
			ds.Comment += r.Comment
			ds.Blank += r.Blank
			ds.Total += r.Total

			if d == filepath.Dir(d) {
				break
			}
		}
	}

	return dirs
}

// AggregateLangs groups file results by language, sorted by code lines.
func AggregateLangs(results []types.FileResult) []types.LangSummary {
	langs := make(map[string]*types.LangSummary)

	for _, r := range results {
		if r.Error != "" {
			continue
		}
		ls, ok := langs[r.Language]
		if !ok {
			ls = &types.LangSummary{Language: r.Language}
			langs[r.Language] = ls
		}
		ls.Files++
		ls.Code += r.Code
		ls.Comment += r.Comment
		ls.Blank += r.Blank
		ls.Total += r.Total
	}

	summaries := make([]types.LangSummary, 0, len(langs))
	for _, ls := range langs {
		summaries = append(summaries, *ls)
	}
	slices.SortFunc(summaries, func(a, b types.LangSummary) int {
		return cmp.Or(
			cmp.Compare(b.Code, a.Code),
			cmp.Compare(b.Total, a.Total),
			strings.Compare(a.Language, b.Language),
		)
	})
	return summaries
}

// metric picks the primary sort value for --sort.
func metric(sortBy string, code, total, files int) int {
	switch sortBy {
	case "total":
		return total
	case "files":
		return files
	}
	return code
}

// TopKFiles returns the top K files sorted by the specified field.
// "files" has no per-file meaning and sorts by total lines.
func TopKFiles(results []types.FileResult, k int, sortBy string) []types.FileResult {
	valid := make([]types.FileResult, 0, len(results))
	for _, r := range results {
		if r.Error == "" {
			valid = append(valid, r)
		}
	}
	slices.SortFunc(valid, func(a, b types.FileResult) int {
		return cmp.Or(
			cmp.Compare(metric(sortBy, b.Code, b.Total, b.Total), metric(sortBy, a.Code, a.Total, a.Total)),
			cmp.Compare(b.Total, a.Total),
			strings.Compare(a.Path, b.Path),
		)
	})
	return valid[:min(k, len(valid))]
}

// TopKDirs returns the top K directories sorted by the specified field.
// The scan root is omitted since it duplicates the overall summary.
func TopKDirs(dirStats map[string]*types.DirStats, k int, sortBy string) []types.DirStats {
	dirs := make([]types.DirStats, 0, len(dirStats))
	for _, ds := range dirStats {
		if ds.Path != "." {
			dirs = append(dirs, *ds)
		}
	}
	slices.SortFunc(dirs, func(a, b types.DirStats) int {
		return cmp.Or(
			cmp.Compare(metric(sortBy, b.Code, b.Total, b.Files), metric(sortBy, a.Code, a.Total, a.Files)),
			cmp.Compare(b.Total, a.Total),
			strings.Compare(a.Path, b.Path),
		)
	})
	return dirs[:min(k, len(dirs))]
}

// SummaryTotals computes overall scan summary.
func SummaryTotals(results []types.FileResult, dirStats map[string]*types.DirStats, langCount int) types.ScanSummary {
	s := types.ScanSummary{
		Languages:   langCount,
		Directories: len(dirStats),
	}

	for _, r := range results {
		if r.Error != "" {
			s.Errors++
			continue
		}
		s.TotalFiles++
		s.TotalCode += r.Code
		s.TotalComment += r.Comment
		s.TotalBlank += r.Blank
		s.TotalLines += r.Total
	}

	return s
}
