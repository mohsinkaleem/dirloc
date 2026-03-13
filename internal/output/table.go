package output

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/dirloc/dirloc/pkg/types"
)

// RenderTable renders results as CLI tables.
func RenderTable(summary types.ScanSummary, topFiles []types.FileResult, topDirs []types.DirStats, langSummaries []types.LangSummary, config types.ScanConfig, elapsed time.Duration) {
	fmt.Printf("\ndirloc \u2014 scanned %d files (%d languages) in %d directories [%s]\n\n",
		summary.TotalFiles, summary.Languages, summary.Directories, formatDuration(elapsed))

	if !config.NoTopFiles && len(topFiles) > 0 {
		renderTopFiles(topFiles, config)
	}

	if !config.NoTopDirs && len(topDirs) > 0 {
		renderTopDirs(topDirs, config)
	}

	if config.ShowLang && len(langSummaries) > 0 {
		renderLangBreakdown(langSummaries)
	}

	// Grand summary line
	fmt.Printf("%d files | %s code | %s comments | %s blank | %s total\n",
		summary.TotalFiles,
		formatNum(summary.TotalCode),
		formatNum(summary.TotalComment),
		formatNum(summary.TotalBlank),
		formatNum(summary.TotalLines))

	if summary.Errors > 0 {
		fmt.Printf("%d files skipped due to errors\n", summary.Errors)
	}
	fmt.Println()
}

func renderTopFiles(files []types.FileResult, config types.ScanConfig) {
	fmt.Printf("Top %d Files by %s Lines\n", len(files), sortLabel(config.SortBy))

	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"Rank", "File", "Language", "Code"}
	if config.ShowLang {
		header = append(header, "Comment", "Blank")
	}
	header = append(header, "Total")
	if config.ShowComplexity {
		header = append(header, "Complexity")
	}
	table.SetHeader(header)
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetColumnAlignment(alignments(len(header)))

	for i, f := range files {
		row := []string{strconv.Itoa(i + 1), f.Path, f.Language, formatNum(f.Code)}
		if config.ShowLang {
			row = append(row, formatNum(f.Comment), formatNum(f.Blank))
		}
		row = append(row, formatNum(f.Total))
		if config.ShowComplexity {
			row = append(row, formatNum(f.Complexity))
		}
		table.Append(row)
	}

	table.Render()
	fmt.Println()
}

func renderTopDirs(dirs []types.DirStats, config types.ScanConfig) {
	fmt.Printf("Top %d Directories by %s Lines\n", len(dirs), sortLabel(config.SortBy))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Rank", "Directory", "Files", "Code", "Total"})
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
	})

	for i, d := range dirs {
		table.Append([]string{
			strconv.Itoa(i + 1),
			d.Path + "/",
			formatNum(d.Files),
			formatNum(d.Code),
			formatNum(d.Total),
		})
	}

	table.Render()
	fmt.Println()
}

func renderLangBreakdown(langs []types.LangSummary) {
	fmt.Println("Language Breakdown")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Language", "Files", "Code", "Comment", "Blank", "Total"})
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
	})

	totalFiles, totalCode, totalComment, totalBlank, totalLines := 0, 0, 0, 0, 0
	for _, l := range langs {
		table.Append([]string{
			l.Language,
			formatNum(l.Files),
			formatNum(l.Code),
			formatNum(l.Comment),
			formatNum(l.Blank),
			formatNum(l.Total),
		})
		totalFiles += l.Files
		totalCode += l.Code
		totalComment += l.Comment
		totalBlank += l.Blank
		totalLines += l.Total
	}

	table.SetFooter([]string{
		"SUM",
		formatNum(totalFiles),
		formatNum(totalCode),
		formatNum(totalComment),
		formatNum(totalBlank),
		formatNum(totalLines),
	})
	table.Render()
	fmt.Println()
}

func sortLabel(sortBy string) string {
	switch sortBy {
	case "total":
		return "Total"
	case "files":
		return "Files"
	default:
		return "Code"
	}
}

func formatNum(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	s := strconv.Itoa(n)
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.0f\u00b5s", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func alignments(n int) []int {
	a := make([]int, n)
	a[0] = tablewriter.ALIGN_RIGHT
	if n > 1 {
		a[1] = tablewriter.ALIGN_LEFT // File path
	}
	if n > 2 {
		a[2] = tablewriter.ALIGN_LEFT // Language
	}
	for i := 3; i < n; i++ {
		a[i] = tablewriter.ALIGN_RIGHT
	}
	return a
}
