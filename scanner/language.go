package scanner

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// UnknownLanguage is returned when a file cannot be mapped to a language.
const UnknownLanguage = "Unknown"

type languageDB struct {
	Extensions    map[string]string   `json:"extensions"`
	Filenames     map[string]string   `json:"filenames"`
	Interpreters  map[string]string   `json:"interpreters"`
	DocLanguages  []string            `json:"docLanguages"`
	Comments      map[string][]string `json:"comments"`
	BlockComments map[string][]string `json:"blockComments"`

	docs map[string]bool
}

var langDB languageDB

// InitLanguages loads the language database from embedded JSON data.
// Must be called before any other functions in this package.
func InitLanguages(data []byte) {
	if err := json.Unmarshal(data, &langDB); err != nil {
		panic("dirloc: failed to parse embedded languages.json: " + err.Error())
	}
	langDB.docs = make(map[string]bool, len(langDB.DocLanguages))
	for _, l := range langDB.DocLanguages {
		langDB.docs[l] = true
	}
}

// DetectLanguage returns the language name for a given file path.
// It checks exact filename first, then extension. Returns UnknownLanguage if unrecognized.
func DetectLanguage(path string) string {
	base := filepath.Base(path)

	if lang, ok := langDB.Filenames[base]; ok {
		return lang
	}

	ext := strings.ToLower(filepath.Ext(base))
	if lang, ok := langDB.Extensions[ext]; ok {
		return lang
	}

	return UnknownLanguage
}

// DetectFileLanguage is DetectLanguage plus a shebang check for extensionless files.
func DetectFileLanguage(path string) string {
	lang := DetectLanguage(path)
	if lang != UnknownLanguage || filepath.Ext(path) != "" {
		return lang
	}
	f, err := os.Open(path)
	if err != nil {
		return UnknownLanguage
	}
	defer f.Close()
	var head [128]byte
	n, _ := io.ReadFull(f, head[:])
	return shebangLanguage(head[:n])
}

// shebangLanguage maps a "#!" line such as "#!/usr/bin/env python3" to a language.
func shebangLanguage(head []byte) string {
	line, ok := bytes.CutPrefix(head, []byte("#!"))
	if !ok {
		return UnknownLanguage
	}
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(string(line))
	if len(fields) == 0 {
		return UnknownLanguage
	}
	name := filepath.Base(fields[0])
	if name == "env" {
		name = ""
		for _, f := range fields[1:] {
			if !strings.HasPrefix(f, "-") {
				name = f
				break
			}
		}
	}
	// python3.12 -> python, perl5 -> perl
	name = strings.TrimRight(name, "0123456789.")
	if lang, ok := langDB.Interpreters[name]; ok {
		return lang
	}
	return UnknownLanguage
}

// IsDocLanguage reports whether lang is a documentation/data format (Markdown, JSON, YAML, ...).
func IsDocLanguage(lang string) bool {
	return langDB.docs[lang]
}

// GetCommentPrefixes returns single-line comment prefixes for a language.
func GetCommentPrefixes(lang string) []string {
	if prefixes, ok := langDB.Comments[lang]; ok {
		return prefixes
	}
	return nil
}

// GetBlockCommentDelimiters returns the block comment start/end for a language.
// Returns empty strings if the language has no block comments.
func GetBlockCommentDelimiters(lang string) (string, string) {
	if delims, ok := langDB.BlockComments[lang]; ok && len(delims) == 2 {
		return delims[0], delims[1]
	}
	return "", ""
}

// LanguageInfo describes one supported language.
type LanguageInfo struct {
	Name     string
	Patterns []string // extensions and exact filenames
	Doc      bool
}

// Languages returns all supported languages sorted by name.
func Languages() []LanguageInfo {
	byName := make(map[string][]string)
	for ext, lang := range langDB.Extensions {
		byName[lang] = append(byName[lang], ext)
	}
	for name, lang := range langDB.Filenames {
		byName[lang] = append(byName[lang], name)
	}

	out := make([]LanguageInfo, 0, len(byName))
	for name, patterns := range byName {
		slices.Sort(patterns)
		out = append(out, LanguageInfo{Name: name, Patterns: patterns, Doc: IsDocLanguage(name)})
	}
	slices.SortFunc(out, func(a, b LanguageInfo) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out
}
