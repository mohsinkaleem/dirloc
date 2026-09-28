package output

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const (
	minPathWidth = 16
	minBarWidth  = 6
	maxBarWidth  = 20
	pctWidth     = len("100.0%")
)

// style applies ANSI SGR codes when enabled.
type style bool

func (s style) paint(code, text string) string {
	if !s || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (s style) bold(t string) string { return s.paint("1", t) }
func (s style) dim(t string) string  { return s.paint("2", t) }
func (s style) cyan(t string) string { return s.paint("36", t) }
func (s style) warn(t string) string { return s.paint("33", t) }

func renderTable(w io.Writer, r Report) {
	st := style(r.Color)
	s := r.Summary
	fmt.Fprintf(w, "\n%s — scanned %s files (%s languages) in %s directories %s\n\n",
		st.bold(st.cyan("dirloc")), st.bold(formatNum(s.TotalFiles)), st.bold(strconv.Itoa(s.Languages)),
		st.bold(formatNum(s.Directories)), st.dim("["+formatDuration(r.Elapsed)+"]"))

	width := getTerminalWidth()
	for _, sec := range r.sections() {
		fmt.Fprintln(w, st.bold(sec.title))
		writeTable(w, st, sec, width)
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, summaryLine(r, st.bold))
	if s.Errors > 0 {
		fmt.Fprintln(w, st.warn(formatNum(s.Errors)+" files skipped due to errors"))
	}
	fmt.Fprintln(w)
}

// cell is display text (possibly with ANSI codes) and its visible width.
type cell struct {
	text  string
	width int
}

// writeTable draws sec as an ASCII table no wider than maxWidth where possible,
// shrinking share bars first and then the path column.
func writeTable(w io.Writer, st style, sec section, maxWidth int) {
	n := len(sec.header)
	left := make([]bool, n)
	for _, i := range sec.left {
		left[i] = true
	}
	isBar := make([]bool, n)
	if len(sec.rows) > 0 {
		for i, v := range sec.rows[0] {
			_, isBar[i] = v.(percent)
			left[i] = left[i] || isBar[i] // percentage sits at a fixed spot, bar grows right
		}
	}

	header := make([]string, n)
	for i, h := range sec.header {
		header[i] = strings.ToUpper(h)
	}
	rows := make([][]string, len(sec.rows))
	for i, row := range sec.rows {
		rows[i] = formatRow(row, true)
	}
	var footer []string
	if sec.footer != nil {
		footer = formatRow(sec.footer, true)
	}

	// Measure natural widths.
	widths := make([]int, n)
	for _, cells := range append(append([][]string{header}, rows...), footer) {
		for i, c := range cells {
			widths[i] = max(widths[i], runewidth.StringWidth(c))
		}
	}
	barWidth := maxBarWidth
	for i := range widths {
		if isBar[i] {
			widths[i] = max(widths[i], pctWidth+1+barWidth)
		}
	}

	// Fit to the terminal.
	total := 3*n + 1
	for _, wd := range widths {
		total += wd
	}
	overflow := total - maxWidth
	for i := range widths {
		if isBar[i] && overflow > 0 {
			shrink := min(overflow, barWidth-minBarWidth)
			barWidth -= shrink
			widths[i] -= shrink
			overflow -= shrink
		}
	}
	if p := sec.pathCol; p > 0 && overflow > 0 {
		widths[p] = max(minPathWidth, widths[p]-overflow)
	}

	plain := func(s string) cell { return cell{s, runewidth.StringWidth(s)} }
	build := func(i int, s string, raw any, isFooter bool) cell {
		switch {
		case isBar[i]:
			pct := strings.Repeat(" ", pctWidth-len(s)) + s
			if isFooter {
				return cell{st.bold(pct), pctWidth}
			}
			bar, bw := shareBar(float64(raw.(percent))/100, barWidth)
			return cell{pct + " " + st.cyan(bar), pctWidth + 1 + bw}
		case isFooter:
			return cell{st.bold(s), runewidth.StringWidth(s)}
		case i == sec.pathCol:
			p := truncatePath(s, widths[i])
			dir, base := splitPath(p)
			return cell{st.dim(dir) + base, runewidth.StringWidth(p)}
		case i == 0 && sec.ranked:
			return cell{st.dim(s), runewidth.StringWidth(s)}
		}
		return plain(s)
	}

	rule := make([]string, n)
	for i, wd := range widths {
		rule[i] = strings.Repeat("-", wd+2)
	}
	ruleLine := st.dim("+" + strings.Join(rule, "+") + "+")
	bar := st.dim("|")

	writeRow := func(cells []cell) {
		var b strings.Builder
		b.WriteString(bar)
		for i, c := range cells {
			pad := strings.Repeat(" ", max(0, widths[i]-c.width))
			b.WriteByte(' ')
			if left[i] {
				b.WriteString(c.text + pad)
			} else {
				b.WriteString(pad + c.text)
			}
			b.WriteString(" " + bar)
		}
		fmt.Fprintln(w, b.String())
	}

	headerCells := make([]cell, n)
	for i, h := range header {
		headerCells[i] = cell{st.bold(h), runewidth.StringWidth(h)}
	}

	fmt.Fprintln(w, ruleLine)
	writeRow(headerCells)
	fmt.Fprintln(w, ruleLine)
	for ri, row := range rows {
		cells := make([]cell, n)
		for i, s := range row {
			cells[i] = build(i, s, sec.rows[ri][i], false)
		}
		writeRow(cells)
	}
	if footer != nil {
		fmt.Fprintln(w, ruleLine)
		cells := make([]cell, n)
		for i, s := range footer {
			cells[i] = build(i, s, nil, true)
		}
		writeRow(cells)
	}
	fmt.Fprintln(w, ruleLine)
}

var barEighths = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

// shareBar draws frac (0..1) of width cells using eighth blocks; returns the bar and its width.
func shareBar(frac float64, width int) (string, int) {
	eighths := int(math.Round(frac * float64(width*8)))
	if frac > 0 && eighths == 0 {
		eighths = 1 // keep tiny non-zero shares visible
	}
	eighths = min(max(eighths, 0), width*8)
	full, rem := eighths/8, eighths%8
	cells := full
	if rem > 0 {
		cells++
	}
	return strings.Repeat("█", full) + barEighths[rem], cells
}

// splitPath splits a path into its directory prefix (with separator) and final element.
func splitPath(p string) (dir, base string) {
	sep := string(filepath.Separator)
	trimmed := strings.TrimSuffix(p, sep)
	i := strings.LastIndex(trimmed, sep)
	return p[:i+1], p[i+1:]
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

// truncatePath shortens path to maxWidth display cells by eliding middle directories.
// It keeps the final element, then the top-level directory, then as many trailing and
// leading directories as fit: "scanner/…/sub/counter_test.go".
func truncatePath(path string, maxWidth int) string {
	if maxWidth <= 0 || runewidth.StringWidth(path) <= maxWidth {
		return path
	}
	sep := string(filepath.Separator)
	parts := strings.Split(strings.TrimSuffix(path, sep), sep)
	if strings.HasSuffix(path, sep) {
		parts[len(parts)-1] += sep
	}

	join := func(h, t int) string {
		s := "…" + sep + strings.Join(parts[t:], sep)
		if h > 0 {
			s = strings.Join(parts[:h], sep) + sep + s
		}
		return s
	}
	fits := func(h, t int) bool { return runewidth.StringWidth(join(h, t)) <= maxWidth }

	// parts[:h] and parts[t:] are kept; parts[h:t] (never empty) is elided.
	h, t := 0, len(parts)-1
	if t == 0 || !fits(h, t) {
		// Even the final element alone is too wide: keep its end.
		return runewidth.TruncateLeft(path, runewidth.StringWidth(path)-(maxWidth-1), "…")
	}
	if h+1 < t && fits(1, t) {
		h = 1
	}
	for t-1 > h && fits(h, t-1) {
		t--
	}
	for h+1 < t && fits(h+1, t) {
		h++
	}

	// Spend leftover width on the start of the elided part, unless only a sliver would show.
	head := ""
	if h > 0 {
		head = strings.Join(parts[:h], sep) + sep
	}
	tail := sep + strings.Join(parts[t:], sep)
	mid := "…"
	if avail := maxWidth - runewidth.StringWidth(head) - runewidth.StringWidth(tail); avail >= 4 {
		mid = runewidth.Truncate(strings.Join(parts[h:t], sep), avail, "…")
	}
	return head + mid + tail
}
