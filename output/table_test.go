package output

import (
	"strings"
	"testing"
	"time"
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
	}{
		// No truncation needed
		{"short/path", 20},
		{"exactly10ch", 20},
		// Truncation cases
		{"very/long/deeply/nested/path/to/some/file.go", 20},
		{"google-cloud-sdk/lib/googlecloudsdk/generated_clients/apis/", 40},
		// Edge cases
		{"abc", 3},
		{"abcd", 3},
		{"", 10},
		{"hello", 0},
	}

	for _, tt := range tests {
		got := truncatePath(tt.path, tt.maxWidth)
		if tt.maxWidth > 0 && len(tt.path) > tt.maxWidth {
			// Truncated path display width should not exceed maxWidth
			// Display width = 1 (for …) + len(ASCII suffix)
			// The … character is 3 bytes but 1 display column
			if strings.HasPrefix(got, "…") {
				displayWidth := 1 + len(got) - len("…") // 1 for ellipsis + ASCII suffix len
				if displayWidth > tt.maxWidth {
					t.Errorf("truncatePath(%q, %d): display width %d exceeds max",
						tt.path, tt.maxWidth, displayWidth)
				}
			}
		}
		if tt.maxWidth <= 0 {
			if got != tt.path {
				t.Errorf("truncatePath(%q, %d) = %q, want original path", tt.path, tt.maxWidth, got)
			}
		}
	}
}

func TestTruncatePath_PreservesEnd(t *testing.T) {
	path := "google-cloud-sdk/lib/googlecloudsdk/generated_clients/apis/"
	truncated := truncatePath(path, 40)

	// Should start with ellipsis
	if !strings.HasPrefix(truncated, "…") {
		t.Errorf("truncated path should start with …, got %q", truncated)
	}

	// The most specific part (end) should be preserved
	suffix := "generated_clients/apis/"
	if !strings.HasSuffix(truncated, suffix) {
		t.Errorf("truncated path should end with %q, got %q", suffix, truncated)
	}
}

func TestComputeMaxPathWidth(t *testing.T) {
	tests := []struct {
		termWidth    int
		numOtherCols int
		minExpected  int
	}{
		{120, 3, 65},  // 120 - 45 = 75
		{80, 3, 20},   // 80 - 45 = 35
		{40, 3, 20},   // 40 - 45 = -5, clamped to 20
		{120, 6, 20},  // 120 - 90 = 30
	}

	for _, tt := range tests {
		got := computeMaxPathWidth(tt.termWidth, tt.numOtherCols)
		if got < tt.minExpected {
			t.Errorf("computeMaxPathWidth(%d, %d) = %d, expected >= %d",
				tt.termWidth, tt.numOtherCols, got, tt.minExpected)
		}
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
