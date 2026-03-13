package output

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"

	"github.com/dirloc/dirloc/types"
)

// RenderTable renders results as CLI tables.
func RenderTable(summary types.ScanSummary, topFiles []types.FileResult, topDirs []types.DirStats, langSummaries []types.LangSummary, config types.ScanConfig, elapsed time.Duration) {
	fmt.Printf("\ndirloc — scanned %s files (%d languages) in %s directories [%s]\n\n",
		formatNum(summary.TotalFiles), summary.Languages, formatNum(summary.Directories), formatDuration(elapsed))

	if !config.NoTopFiles && len(topFiles) > 0 {
		renderTopFiles(topFiles, config)
	}

	if !config.NoTopDirs && len(topDirs) > 0 {
		renderTopDirs(topDirs, config)
	}

	if config.ShowLang && len(langSummaries) > 0 {
		renderLangBreakdown(langSummaries)
	}

	// Summary line: detailed breakdown only when --lang is used
	if config.ShowLang {
		fmt.Printf("%s files | %s code | %s comments | %s blank | %s total\n",
			formatNum(summary.TotalFiles),
			formatNum(summary.TotalCode),
			formatNum(summary.TotalComment),
			formatNum(summary.TotalBlank),
			formatNum(summary.TotalLines))
	} else {
		fmt.Printf("%s files | %s total lines\n",
			formatNum(summary.TotalFiles),
			formatNum(summary.TotalLines))
	}

	if summary.Errors > 0 {
		fmt.Printf("%s files skipped due to errors\n", formatNum(summary.Errors))
	}
	fmt.Println()
}

func renderTopFiles(files []types.FileResult, config types.ScanConfig) {
	fmt.Printf("Top %d Files by %s\n", len(files), topFilesLabel(config))

	termWidth := getTerminalWidth()

	header := []string{"Rank", "File", "Language"}
	if config.ShowLang {
		header = append(header, "Code", "Comment", "Blank")
	}
	header = append(header, "Lines")
	if config.ShowComplexity {
		header = append(header, "Complexity")
	}

	maxPath := computeMaxPathWidth(termWidth, len(header)-1)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)

	aligns := make([]int, len(header))
	for i := range aligns {
		aligns[i] = tablewriter.ALIGN_RIGHT
	}
	aligns[1] = tablewriter.ALIGN_LEFT // File
	aligns[2] = tablewriter.ALIGN_LEFT // Language
	table.SetColumnAlignment(aligns)

	for i, f := range files {
		row := []string{
			strconv.Itoa(i + 1),
			truncatePath(f.Path, maxPath),
			f.Language,
		}
		if config.ShowLang {
			row = append(row, formatNum(f.Code), formatNum(f.Comment), formatNum(f.Blank))
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
	fmt.Printf("Top %d Directories by %s\n", len(dirs), topDirsLabel(config))

	termWidth := getTerminalWidth()

	header := []string{"Rank", "Directory", "Files"}
	if config.ShowLang {
		header = append(header, "Code")
	}
	header = append(header, "Lines")

	maxPath := computeMaxPathWidth(termWidth, len(header)-1)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetBorder(true)
	table.SetAlignment(tablewriter.ALIGN_RIGHT)

	aligns := make([]int, len(header))
	for i := range aligns {
		aligns[i] = tablewriter.ALIGN_RIGHT
	}
	aligns[1] = tablewriter.ALIGN_LEFT // Directory
	table.SetColumnAlignment(aligns)

	for i, d := range dirs {
		row := []string{
			strconv.Itoa(i + 1),
			truncatePath(d.Path+"/", maxPath),
			formatNum(d.Files),
		}
		if config.ShowLang {
			row = append(row, formatNum(d.Code))
		}
		row = append(row, formatNum(d.Total))
		table.Append(row)
	}

	table.Render()
	fmt.Println()
}

func renderLangBreakdown(langs []types.LangSummary) {
	fmt.Println("Language Breakdown")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Language", "Files", "Code", "Comment", "Blank", "Lines"})
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

func topFilesLabel(config types.ScanConfig) string {
	if !config.ShowLang || config.SortBy == "total" {
		return "Total Lines"
	}
	return "Code Lines"
}

func topDirsLabel(config types.ScanConfig) string {
	if config.SortBy == "files" {
		return "File Count"
	}
	if !config.ShowLang || config.SortBy == "total" {
		return "Total Lines"
	}
	return "Code Lines"
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
		return fmt.Sprintf("%.0fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 120
	}
	return width
}

// truncatePath shortens a path to maxWidth, preserving the rightmost (most specific) portion.
func truncatePath(path string, maxWidth int) string {
	if len(path) <= maxWidth || maxWidth <= 0 {
		return path
	}
	if maxWidth <= 3 {
		return "…"
	}
	return "…" + path[len(path)-(maxWidth-1):]
}

// computeMaxPathWidth estimates available width for the path column.
func computeMaxPathWidth(termWidth, numOtherCols int) int {
	// Each non-path column takes ~15 chars (content + border/padding)
	otherWidth := numOtherCols * 15
	available := termWidth - otherWidth
	if available < 20 {
		return 20
	}
	return available
}
