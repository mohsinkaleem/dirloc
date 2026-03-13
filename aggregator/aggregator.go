package aggregator

import (
	"path/filepath"
	"sort"

	"github.com/dirloc/dirloc/types"
)

// AggregateDirs groups file results by directory and rolls up stats.
func AggregateDirs(results []types.FileResult) map[string]*types.DirStats {
	dirs := make(map[string]*types.DirStats)

	for _, r := range results {
		if r.Error != "" {
			continue
		}
		dir := filepath.Dir(r.Path)

		// Walk up the directory hierarchy and add stats to each ancestor
		for d := dir; ; d = filepath.Dir(d) {
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

			if d == "." || d == "/" || d == filepath.Dir(d) {
				break
			}
		}
	}

	return dirs
}

// AggregateLangs groups file results by language.
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

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Code > summaries[j].Code
	})

	return summaries
}

// TopKFiles returns the top K files sorted by the specified field.
func TopKFiles(results []types.FileResult, k int, sortBy string) []types.FileResult {
	// Filter out errors
	valid := make([]types.FileResult, 0, len(results))
	for _, r := range results {
		if r.Error == "" {
			valid = append(valid, r)
		}
	}

	sortFunc := fileSortFunc(sortBy)
	sort.Slice(valid, sortFunc(valid))

	if k > len(valid) {
		k = len(valid)
	}
	return valid[:k]
}

// TopKDirs returns the top K directories sorted by the specified field.
func TopKDirs(dirStats map[string]*types.DirStats, k int, sortBy string) []types.DirStats {
	dirs := make([]types.DirStats, 0, len(dirStats))
	for _, ds := range dirStats {
		dirs = append(dirs, *ds)
	}

	sortFunc := dirSortFunc(sortBy)
	sort.Slice(dirs, sortFunc(dirs))

	if k > len(dirs) {
		k = len(dirs)
	}
	return dirs[:k]
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

func fileSortFunc(sortBy string) func([]types.FileResult) func(int, int) bool {
	return func(items []types.FileResult) func(int, int) bool {
		switch sortBy {
		case "total", "files":
			// "files" has no meaningful per-file metric; fall through to "total".
			return func(i, j int) bool {
				if items[i].Total != items[j].Total {
					return items[i].Total > items[j].Total
				}
				return items[i].Path < items[j].Path
			}
		default: // "code"
			return func(i, j int) bool {
				if items[i].Code != items[j].Code {
					return items[i].Code > items[j].Code
				}
				if items[i].Total != items[j].Total {
					return items[i].Total > items[j].Total
				}
				return items[i].Path < items[j].Path
			}
		}
	}
}

func dirSortFunc(sortBy string) func([]types.DirStats) func(int, int) bool {
	return func(items []types.DirStats) func(int, int) bool {
		switch sortBy {
		case "total":
			return func(i, j int) bool {
				if items[i].Total != items[j].Total {
					return items[i].Total > items[j].Total
				}
				return items[i].Path < items[j].Path
			}
		case "files":
			return func(i, j int) bool {
				if items[i].Files != items[j].Files {
					return items[i].Files > items[j].Files
				}
				return items[i].Path < items[j].Path
			}
		default: // "code"
			return func(i, j int) bool {
				if items[i].Code != items[j].Code {
					return items[i].Code > items[j].Code
				}
				if items[i].Total != items[j].Total {
					return items[i].Total > items[j].Total
				}
				return items[i].Path < items[j].Path
			}
		}
	}
}
