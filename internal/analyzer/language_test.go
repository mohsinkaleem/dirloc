package analyzer

import "testing"

func TestDetectLanguage_Extension(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "Go"},
		{"app.py", "Python"},
		{"index.js", "JavaScript"},
		{"style.css", "CSS"},
		{"page.html", "HTML"},
		{"data.json", "JSON"},
		{"script.sh", "Shell"},
		{"app.ts", "TypeScript"},
		{"main.rs", "Rust"},
		{"unknown.xyz", "Unknown"},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.path)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDetectLanguage_Filename(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"Makefile", "Makefile"},
		{"Dockerfile", "Docker"},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.path)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestDetectLanguage_CaseInsensitiveExt(t *testing.T) {
	// Extensions are lowered before lookup
	got := DetectLanguage("file.GO")
	// filepath.Ext returns ".GO", then it's lowered to ".go"
	if got != "Go" {
		t.Errorf("DetectLanguage(file.GO) = %q, want Go", got)
	}
}

func TestGetCommentPrefixes(t *testing.T) {
	tests := []struct {
		lang string
		want int // expected number of prefixes
	}{
		{"Go", 1},      // //
		{"Python", 1},  // #
		{"Unknown", 0}, // none
	}

	for _, tt := range tests {
		got := GetCommentPrefixes(tt.lang)
		if len(got) != tt.want {
			t.Errorf("GetCommentPrefixes(%q) returned %d prefixes, want %d", tt.lang, len(got), tt.want)
		}
	}
}

func TestIsCodeFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"app.py", true},
		{"readme.txt", true},  // .txt maps to Text language
		{"image.png", false},
		{"unknown.xyz", false},
	}

	for _, tt := range tests {
		got := IsCodeFile(tt.path)
		if got != tt.want {
			t.Errorf("IsCodeFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func BenchmarkDetectLanguage(b *testing.B) {
	paths := []string{
		"src/main.go",
		"lib/utils.py",
		"web/index.js",
		"styles/main.css",
		"unknown.xyz",
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		DetectLanguage(paths[i%len(paths)])
	}
}

func BenchmarkIsCodeFile(b *testing.B) {
	paths := []string{
		"main.go",
		"readme.txt",
		"image.png",
		"app.py",
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		IsCodeFile(paths[i%len(paths)])
	}
}
