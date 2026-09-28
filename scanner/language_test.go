package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestShebangLanguage(t *testing.T) {
	tests := []struct {
		head string
		want string
	}{
		{"#!/usr/bin/env python3\nprint(1)\n", "Python"},
		{"#!/usr/bin/python3.12\n", "Python"},
		{"#!/bin/bash -e\n", "Shell"},
		{"#!/usr/bin/env -S deno run\n", "TypeScript"},
		{"#!/usr/bin/env node\r\n", "JavaScript"},
		{"#!/usr/bin/perl5\n", "Perl"},
		{"#!/usr/bin/unknown-tool\n", UnknownLanguage},
		{"#!\n", UnknownLanguage},
		{"no shebang\n", UnknownLanguage},
		{"", UnknownLanguage},
	}
	for _, tt := range tests {
		if got := shebangLanguage([]byte(tt.head)); got != tt.want {
			t.Errorf("shebangLanguage(%q) = %q, want %q", tt.head, got, tt.want)
		}
	}
}

func TestDetectFileLanguage_Shebang(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "deploy")
	os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0755)
	plain := filepath.Join(dir, "LICENSE")
	os.WriteFile(plain, []byte("MIT License\n"), 0644)
	// Extension-based files are never sniffed.
	noSniff := filepath.Join(dir, "notes.xyz")
	os.WriteFile(noSniff, []byte("#!/bin/sh\n"), 0644)

	if got := DetectFileLanguage(script); got != "Shell" {
		t.Errorf("script: got %q, want Shell", got)
	}
	if got := DetectFileLanguage(plain); got != UnknownLanguage {
		t.Errorf("LICENSE: got %q, want %q", got, UnknownLanguage)
	}
	if got := DetectFileLanguage(noSniff); got != UnknownLanguage {
		t.Errorf("notes.xyz: got %q, want %q", got, UnknownLanguage)
	}
}

func TestIsDocLanguage(t *testing.T) {
	for _, l := range []string{"Markdown", "JSON", "YAML", "Text"} {
		if !IsDocLanguage(l) {
			t.Errorf("IsDocLanguage(%q) = false, want true", l)
		}
	}
	for _, l := range []string{"Go", "Python", UnknownLanguage} {
		if IsDocLanguage(l) {
			t.Errorf("IsDocLanguage(%q) = true, want false", l)
		}
	}
}

func TestLanguages(t *testing.T) {
	langs := Languages()
	if len(langs) < 50 {
		t.Fatalf("expected 50+ languages, got %d", len(langs))
	}
	for _, l := range langs {
		if l.Name == "Go" {
			if len(l.Patterns) != 1 || l.Patterns[0] != ".go" || l.Doc {
				t.Errorf("unexpected Go entry: %+v", l)
			}
			return
		}
	}
	t.Error("Go not found in Languages()")
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
