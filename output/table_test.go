package output

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func TestFormatNum(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{12345678, "12,345,678"},
		{123456789, "123,456,789"},
		{1000000000, "1,000,000,000"},
	}

	for _, tt := range tests {
		got := formatNum(tt.input)
		if got != tt.want {
			t.Errorf("formatNum(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{500 * time.Microsecond, "500µs"},
		{100 * time.Millisecond, "100ms"},
		{1500 * time.Millisecond, "1.50s"},
		{2*time.Second + 500*time.Millisecond, "2.50s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		path     string
		maxWidth int
		want     string
	}{
		// No truncation needed
		{"short/path", 20, "short/path"},
		{"", 10, ""},
		{"hello", 0, "hello"},
		{"abc", 3, "abc"},
		// Middle directories are elided; top-level dir and file name are kept.
		{"scanner/sub/deep/counter_test.go", 25, "scanner/…/counter_test.go"},
		{"very/long/deeply/nested/path/to/some/file.go", 20, "very/…/some/file.go"},
		{"a/very/long/directory/name/", 15, "a/very/…/name/"},
		// Leftover width shows the start of the elided part.
		{"aggregator/aggregator_test.go", 27, "aggrega…/aggregator_test.go"},
		{"google-cloud-sdk/lib/googlecloudsdk/generated_clients/apis/", 40, "google-cloud-sdk/lib/googleclouds…/apis/"},
		// A single element that is too wide keeps its end.
		{"abcd", 3, "…cd"},
		// Wide (CJK) characters count as two cells.
		{"文档/说明文件.go", 12, "…说明文件.go"},
	}

	for _, tt := range tests {
		path, want := filepath.FromSlash(tt.path), filepath.FromSlash(tt.want)
		got := truncatePath(path, tt.maxWidth)
		if got != want {
			t.Errorf("truncatePath(%q, %d) = %q, want %q", path, tt.maxWidth, got, want)
		}
		if tt.maxWidth > 0 && runewidth.StringWidth(got) > tt.maxWidth {
			t.Errorf("truncatePath(%q, %d) width %d exceeds max", path, tt.maxWidth, runewidth.StringWidth(got))
		}
	}
}

func TestShareBar(t *testing.T) {
	tests := []struct {
		frac  float64
		width int
		want  string
	}{
		{0, 10, ""},
		{0.001, 10, "▏"}, // tiny non-zero shares stay visible
		{0.5, 10, "█████"},
		{0.55, 10, "█████▌"},
		{1, 10, "██████████"},
		{1.5, 4, "████"},
	}
	for _, tt := range tests {
		got, w := shareBar(tt.frac, tt.width)
		if got != tt.want || w != utf8.RuneCountInString(tt.want) {
			t.Errorf("shareBar(%v, %d) = %q (%d), want %q", tt.frac, tt.width, got, w, tt.want)
		}
	}
}

func TestWriteTable_FitsWidth(t *testing.T) {
	long := filepath.FromSlash("services/payments/internal/handlers/refunds/refund.go")
	sec := section{
		header:  []string{"Rank", "Directory", "Files", "Lines", "Share"},
		left:    []int{1},
		pathCol: 1,
		ranked:  true,
		rows: [][]any{
			{1, long, 12, 34567, percent(61.6)},
			{2, "cmd/", 2, 300, percent(0.5)},
		},
	}

	for _, width := range []int{60, 80, 120} {
		var buf bytes.Buffer
		writeTable(&buf, style(false), sec, width)
		out := buf.String()
		if strings.Contains(out, "\x1b[") {
			t.Fatal("unexpected ANSI codes with colour disabled")
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if w := runewidth.StringWidth(line); w > width {
				t.Errorf("width %d: line is %d wide: %q", width, w, line)
			}
		}
		if !strings.Contains(out, "refund.go") || !strings.Contains(out, "34,567") || !strings.Contains(out, " 61.6% ") {
			t.Errorf("width %d: missing expected content:\n%s", width, out)
		}
	}

	var buf bytes.Buffer
	writeTable(&buf, style(true), sec, 80)
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Error("expected ANSI codes with colour enabled")
	}
}

func BenchmarkFormatNum(b *testing.B) {
	nums := []int{0, 42, 999, 1234, 123456, 1234567, 123456789}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		formatNum(nums[i%len(nums)])
	}
}

func BenchmarkTruncatePath(b *testing.B) {
	path := "google-cloud-sdk/lib/googlecloudsdk/generated_clients/apis/compute_v1/resources.py"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		truncatePath(path, 50)
	}
}
