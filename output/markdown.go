package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/mohsinkaleem/dirloc/types"
)

// maxPieSlices caps the Mermaid pie chart; smaller languages are grouped as "Other".
const maxPieSlices = 8

func renderMarkdown(w io.Writer, r Report) {
	s := r.Summary
	fmt.Fprint(w, "# dirloc Report\n\n")
	fmt.Fprintf(w, "Scanned **%s** files (**%d** languages) in **%s** directories [%s]\n\n",
		formatNum(s.TotalFiles), s.Languages, formatNum(s.Directories), formatDuration(r.Elapsed))

	for _, sec := range r.sections() {
		fmt.Fprintf(w, "## %s\n\n", sec.title)
		writeMarkdownRow(w, sec.header)

		align := make([]string, len(sec.header))
		for i := range align {
			align[i] = "---:"
		}
		for _, i := range sec.left {
			align[i] = ":---"
		}
		writeMarkdownRow(w, align)

		for _, row := range sec.rows {
			writeMarkdownRow(w, formatRow(row, true))
		}
		if sec.footer != nil {
			writeMarkdownRow(w, formatRow(sec.footer, true))
		}
		fmt.Fprintln(w)
	}

	if r.Config.ShowLang {
		writeMermaidPie(w, r.Langs)
	}

	fmt.Fprintf(w, "**Summary:** %s\n\n", summaryLine(r, func(s string) string { return s }))
	if s.Errors > 0 {
		fmt.Fprintf(w, "_%s files skipped due to errors_\n\n", formatNum(s.Errors))
	}
}

func writeMarkdownRow(w io.Writer, cells []string) {
	escaped := make([]string, len(cells))
	for i, c := range cells {
		escaped[i] = strings.ReplaceAll(c, "|", `\|`)
	}
	fmt.Fprintf(w, "| %s |\n", strings.Join(escaped, " | "))
}

// writeMermaidPie renders code lines per language as a Mermaid pie chart (rendered by GitHub).
func writeMermaidPie(w io.Writer, langs []types.LangSummary) {
	var shown []types.LangSummary
	other := 0
	for _, l := range langs {
		switch {
		case l.Code == 0:
		case len(shown) < maxPieSlices:
			shown = append(shown, l)
		default:
			other += l.Code
		}
	}
	if len(shown) == 0 {
		return
	}

	fmt.Fprint(w, "```mermaid\npie showData title Code lines by language\n")
	for _, l := range shown {
		fmt.Fprintf(w, "    \"%s\" : %d\n", strings.ReplaceAll(l.Language, `"`, "'"), l.Code)
	}
	if other > 0 {
		fmt.Fprintf(w, "    \"Other\" : %d\n", other)
	}
	fmt.Fprint(w, "```\n\n")
}
