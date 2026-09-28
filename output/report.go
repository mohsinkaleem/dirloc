package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/term"

	"github.com/mohsinkaleem/dirloc/types"
)

// Formats lists the supported --format values.
var Formats = []string{"table", "json", "md", "csv"}

// Report holds everything a renderer needs.
type Report struct {
	Summary types.ScanSummary
	Files   []types.FileResult
	Dirs    []types.DirStats
	Langs   []types.LangSummary
	Config  types.ScanConfig
	Elapsed time.Duration
	Color   bool // ANSI colours in table output
}

// ColorEnabled reports whether table output to f should use ANSI colours.
// It honours NO_COLOR (https://no-color.org) and TERM=dumb.
func ColorEnabled(f *os.File, disabled bool) bool {
	if disabled || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Render writes the report to w in r.Config.Format.
func Render(w io.Writer, r Report) error {
	switch r.Config.Format {
	case "json":
		return renderJSON(w, r)
	case "md":
		renderMarkdown(w, r)
	case "csv":
		return renderCSV(w, r)
	default:
		renderTable(w, r)
	}
	return nil
}

// percent renders as "12.3%" in human output and "12.3" in CSV; the table adds a bar.
type percent float64

// section is one titled table shared by the table, Markdown and CSV renderers.
// Cells are string, int or percent so each renderer can format numbers its own way.
type section struct {
	title   string
	header  []string
	left    []int // left-aligned column indexes
	pathCol int   // path column truncated to terminal width; 0 = none
	ranked  bool  // first column is a rank
	rows    [][]any
	footer  []any
}

func (r Report) sections() []section {
	var out []section
	if !r.Config.NoTopFiles && len(r.Files) > 0 {
		out = append(out, filesSection(r.Files, r.Config))
	}
	if !r.Config.NoTopDirs && len(r.Dirs) > 0 {
		out = append(out, dirsSection(r))
	}
	if r.Config.ShowLang && len(r.Langs) > 0 {
		out = append(out, langsSection(r.Langs))
	}
	return out
}

func filesSection(files []types.FileResult, cfg types.ScanConfig) section {
	s := section{
		title:   fmt.Sprintf("Top %d Files by %s", len(files), topFilesLabel(cfg)),
		header:  []string{"Rank", "File", "Language"},
		left:    []int{1, 2},
		pathCol: 1,
		ranked:  true,
	}
	if cfg.ShowLang {
		s.header = append(s.header, "Code", "Comment", "Blank")
	}
	s.header = append(s.header, "Lines")
	if cfg.ShowComplexity {
		s.header = append(s.header, "Complexity")
	}

	for i, f := range files {
		row := []any{i + 1, f.Path, f.Language}
		if cfg.ShowLang {
			row = append(row, f.Code, f.Comment, f.Blank)
		}
		row = append(row, f.Total)
		if cfg.ShowComplexity {
			row = append(row, f.Complexity)
		}
		s.rows = append(s.rows, row)
	}
	return s
}

func dirsSection(r Report) section {
	cfg := r.Config
	s := section{
		title:   fmt.Sprintf("Top %d Directories by %s", len(r.Dirs), topDirsLabel(cfg)),
		header:  []string{"Rank", "Directory", "Files"},
		left:    []int{1},
		pathCol: 1,
		ranked:  true,
	}
	if cfg.ShowLang {
		s.header = append(s.header, "Code")
	}
	s.header = append(s.header, "Lines", "Share")

	for i, d := range r.Dirs {
		row := []any{i + 1, d.Path + string(filepath.Separator), d.Files}
		if cfg.ShowLang {
			row = append(row, d.Code)
		}
		row = append(row, d.Total, dirShare(d, r.Summary, cfg))
		s.rows = append(s.rows, row)
	}
	return s
}

// dirShare is the directory's share of the metric named in the section title.
func dirShare(d types.DirStats, s types.ScanSummary, cfg types.ScanConfig) percent {
	part, whole := d.Total, s.TotalLines
	switch {
	case cfg.SortBy == "files":
		part, whole = d.Files, s.TotalFiles
	case cfg.ShowLang && cfg.SortBy == "code":
		part, whole = d.Code, s.TotalCode
	}
	if whole == 0 {
		return 0
	}
	return percent(float64(part) * 100 / float64(whole))
}

func langsSection(langs []types.LangSummary) section {
	s := section{
		title:  "Language Breakdown",
		header: []string{"Language", "Files", "Code", "Comment", "Blank", "Lines", "Code %"},
		left:   []int{0},
	}

	var sum types.LangSummary
	for _, l := range langs {
		sum.Files += l.Files
		sum.Code += l.Code
		sum.Comment += l.Comment
		sum.Blank += l.Blank
		sum.Total += l.Total
	}

	share := func(code int) percent {
		if sum.Code == 0 {
			return 0
		}
		return percent(float64(code) * 100 / float64(sum.Code))
	}
	for _, l := range langs {
		s.rows = append(s.rows, []any{l.Language, l.Files, l.Code, l.Comment, l.Blank, l.Total, share(l.Code)})
	}
	s.footer = []any{"SUM", sum.Files, sum.Code, sum.Comment, sum.Blank, sum.Total, share(sum.Code)}
	return s
}

// formatRow converts cells to strings; human adds thousands separators and "%".
func formatRow(row []any, human bool) []string {
	out := make([]string, len(row))
	for i, v := range row {
		switch v := v.(type) {
		case int:
			if human {
				out[i] = formatNum(v)
			} else {
				out[i] = strconv.Itoa(v)
			}
		case percent:
			out[i] = strconv.FormatFloat(float64(v), 'f', 1, 64)
			if human {
				out[i] += "%"
			}
		default:
			out[i] = fmt.Sprint(v)
		}
	}
	return out
}

// summaryLine renders the totals line; num styles each number (identity for plain text).
func summaryLine(r Report, num func(string) string) string {
	s := r.Summary
	if r.Config.ShowLang {
		return fmt.Sprintf("%s files | %s code | %s comments | %s blank | %s total",
			num(formatNum(s.TotalFiles)), num(formatNum(s.TotalCode)), num(formatNum(s.TotalComment)),
			num(formatNum(s.TotalBlank)), num(formatNum(s.TotalLines)))
	}
	return fmt.Sprintf("%s files | %s total lines", num(formatNum(s.TotalFiles)), num(formatNum(s.TotalLines)))
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
	return topFilesLabel(config)
}

// renderCSV writes one CSV block per section, separated by a blank line.
func renderCSV(w io.Writer, r Report) error {
	cw := csv.NewWriter(w)
	for i, s := range r.sections() {
		if i > 0 {
			cw.Write(nil)
		}
		cw.Write(s.header)
		for _, row := range s.rows {
			cw.Write(formatRow(row, false))
		}
	}
	cw.Flush()
	return cw.Error()
}
