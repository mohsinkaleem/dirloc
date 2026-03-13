package output

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dirloc/dirloc/pkg/types"
)

// RenderMarkdown outputs scan results as Markdown tables.
func RenderMarkdown(summary types.ScanSummary, topFiles []types.FileResult, topDirs []types.DirStats, langSummaries []types.LangSummary, config types.ScanConfig, elapsed time.Duration) {
	fmt.Printf("# dirloc Report\n\n")
	fmt.Printf("Scanned **%d** files (**%d** languages) in **%d** directories [%s]\n\n",
		summary.TotalFiles, summary.Languages, summary.Directories, formatDuration(elapsed))

	if !config.NoTopFiles && len(topFiles) > 0 {
		renderTopFilesMarkdown(topFiles, config)
	}

	if !config.NoTopDirs && len(topDirs) > 0 {
		renderTopDirsMarkdown(topDirs, config)
	}

	if config.ShowLang && len(langSummaries) > 0 {
		renderLangMarkdown(langSummaries)
	}

	fmt.Printf("**Summary:** %d files | %s code | %s comments | %s blank | %s total\n\n",
		summary.TotalFiles,
		formatNum(summary.TotalCode),
		formatNum(summary.TotalComment),
		formatNum(summary.TotalBlank),
		formatNum(summary.TotalLines))

	if summary.Errors > 0 {
		fmt.Printf("_%d files skipped due to errors_\n\n", summary.Errors)
	}
}

func renderTopFilesMarkdown(files []types.FileResult, config types.ScanConfig) {
	fmt.Printf("## Top %d Files by %s Lines\n\n", len(files), sortLabel(config.SortBy))
	fmt.Print("| Rank | File | Language | Code |")
	if config.ShowLang {
		fmt.Print(" Comment | Blank |")
	}
	fmt.Print(" Total |")
	if config.ShowComplexity {
		fmt.Print(" Complexity |")
	}
	fmt.Println()

	fmt.Print("|---:|:---|:---|---:|")
	if config.ShowLang {
		fmt.Print("---:|---:|")
	}
	fmt.Print("---:|")
	if config.ShowComplexity {
		fmt.Print("---:|")
	}
	fmt.Println()

	for i, f := range files {
		fmt.Printf("| %d | %s | %s | %s |", i+1, f.Path, f.Language, formatNum(f.Code))
		if config.ShowLang {
			fmt.Printf(" %s | %s |", formatNum(f.Comment), formatNum(f.Blank))
		}
		fmt.Printf(" %s |", formatNum(f.Total))
		if config.ShowComplexity {
			fmt.Printf(" %s |", strconv.Itoa(f.Complexity))
		}
		fmt.Println()
	}
	fmt.Println()
}

func renderTopDirsMarkdown(dirs []types.DirStats, config types.ScanConfig) {
	fmt.Printf("## Top %d Directories by %s Lines\n\n", len(dirs), sortLabel(config.SortBy))
	fmt.Println("| Rank | Directory | Files | Code | Total |")
	fmt.Println("|---:|:---|---:|---:|---:|")

	for i, d := range dirs {
		fmt.Printf("| %d | %s/ | %s | %s | %s |\n",
			i+1, d.Path, formatNum(d.Files), formatNum(d.Code), formatNum(d.Total))
	}
	fmt.Println()
}

func renderLangMarkdown(langs []types.LangSummary) {
	fmt.Println("## Language Breakdown")
	fmt.Println()
	fmt.Println("| Language | Files | Code | Comment | Blank | Total |")
	fmt.Println("|:---|---:|---:|---:|---:|---:|")

	for _, l := range langs {
		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
			l.Language, formatNum(l.Files), formatNum(l.Code),
			formatNum(l.Comment), formatNum(l.Blank), formatNum(l.Total))
	}
	fmt.Println()
}
