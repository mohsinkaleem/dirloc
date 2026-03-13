package output

import (
	"fmt"
	"time"

	"github.com/dirloc/dirloc/pkg/types"
)

// RenderMarkdown outputs scan results as Markdown tables.
func RenderMarkdown(summary types.ScanSummary, topFiles []types.FileResult, topDirs []types.DirStats, langSummaries []types.LangSummary, config types.ScanConfig, elapsed time.Duration) {
	fmt.Printf("# dirloc Report\n\n")
	fmt.Printf("Scanned **%s** files (**%d** languages) in **%s** directories [%s]\n\n",
		formatNum(summary.TotalFiles), summary.Languages, formatNum(summary.Directories), formatDuration(elapsed))

	if !config.NoTopFiles && len(topFiles) > 0 {
		renderTopFilesMarkdown(topFiles, config)
	}

	if !config.NoTopDirs && len(topDirs) > 0 {
		renderTopDirsMarkdown(topDirs, config)
	}

	if config.ShowLang && len(langSummaries) > 0 {
		renderLangMarkdown(langSummaries)
	}

	if config.ShowLang {
		fmt.Printf("**Summary:** %s files | %s code | %s comments | %s blank | %s total\n\n",
			formatNum(summary.TotalFiles),
			formatNum(summary.TotalCode),
			formatNum(summary.TotalComment),
			formatNum(summary.TotalBlank),
			formatNum(summary.TotalLines))
	} else {
		fmt.Printf("**Summary:** %s files | %s total lines\n\n",
			formatNum(summary.TotalFiles),
			formatNum(summary.TotalLines))
	}

	if summary.Errors > 0 {
		fmt.Printf("_%s files skipped due to errors_\n\n", formatNum(summary.Errors))
	}
}

func renderTopFilesMarkdown(files []types.FileResult, config types.ScanConfig) {
	fmt.Printf("## Top %d Files by %s\n\n", len(files), topFilesLabel(config))

	// Header
	fmt.Print("| Rank | File | Language |")
	if config.ShowLang {
		fmt.Print(" Code | Comment | Blank |")
	}
	fmt.Print(" Total |")
	if config.ShowComplexity {
		fmt.Print(" Complexity |")
	}
	fmt.Println()

	// Alignment
	fmt.Print("|---:|:---|:---|")
	if config.ShowLang {
		fmt.Print("---:|---:|---:|")
	}
	fmt.Print("---:|")
	if config.ShowComplexity {
		fmt.Print("---:|")
	}
	fmt.Println()

	// Rows
	for i, f := range files {
		fmt.Printf("| %d | %s | %s |", i+1, f.Path, f.Language)
		if config.ShowLang {
			fmt.Printf(" %s | %s | %s |", formatNum(f.Code), formatNum(f.Comment), formatNum(f.Blank))
		}
		fmt.Printf(" %s |", formatNum(f.Total))
		if config.ShowComplexity {
			fmt.Printf(" %s |", formatNum(f.Complexity))
		}
		fmt.Println()
	}
	fmt.Println()
}

func renderTopDirsMarkdown(dirs []types.DirStats, config types.ScanConfig) {
	fmt.Printf("## Top %d Directories by %s\n\n", len(dirs), topDirsLabel(config))

	// Header
	fmt.Print("| Rank | Directory | Files |")
	if config.ShowLang {
		fmt.Print(" Code |")
	}
	fmt.Println(" Total |")

	// Alignment
	fmt.Print("|---:|:---|---:|")
	if config.ShowLang {
		fmt.Print("---:|")
	}
	fmt.Println("---:|")

	// Rows
	for i, d := range dirs {
		fmt.Printf("| %d | %s/ | %s |", i+1, d.Path, formatNum(d.Files))
		if config.ShowLang {
			fmt.Printf(" %s |", formatNum(d.Code))
		}
		fmt.Printf(" %s |\n", formatNum(d.Total))
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
