package cmd

import "testing"

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		err   bool
	}{
		// Basic sizes
		{"10MB", 10 * 1024 * 1024, false},
		{"10mb", 10 * 1024 * 1024, false},
		{"500KB", 500 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"100B", 100, false},

		// Decimal sizes
		{"1.5MB", int64(1.5 * 1024 * 1024), false},
		{"2.5GB", int64(2.5 * 1024 * 1024 * 1024), false},

		// Plain numbers
		{"1024", 1024, false},

		// Edge cases
		{"0", 0, false},
		{"", 0, false},

		// Errors
		{"abc", 0, true},
		{"10XB", 0, true},
		{"MB", 0, true},
	}

	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if tt.err && err == nil {
			t.Errorf("parseSize(%q): expected error", tt.input)
			continue
		}
		if !tt.err && err != nil {
			t.Errorf("parseSize(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func BenchmarkParseSize(b *testing.B) {
	sizes := []string{"10MB", "500KB", "1.5GB", "1024", "100B"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parseSize(sizes[i%len(sizes)])
	}
}
