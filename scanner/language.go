package scanner

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

type languageDB struct {
	Extensions    map[string]string   `json:"extensions"`
	Filenames     map[string]string   `json:"filenames"`
	Comments      map[string][]string `json:"comments"`
	BlockComments map[string][]string `json:"blockComments"`
}

var langDB languageDB

// InitLanguages loads the language database from embedded JSON data.
// Must be called before any other functions in this package.
func InitLanguages(data []byte) {
	if err := json.Unmarshal(data, &langDB); err != nil {
		panic("dirloc: failed to parse embedded languages.json: " + err.Error())
	}
}

// DetectLanguage returns the language name for a given file path.
// It checks exact filename first, then extension. Returns "Unknown" if unrecognized.
func DetectLanguage(path string) string {
	base := filepath.Base(path)

	if lang, ok := langDB.Filenames[base]; ok {
		return lang
	}

	ext := strings.ToLower(filepath.Ext(base))
	if lang, ok := langDB.Extensions[ext]; ok {
		return lang
	}

	return "Unknown"
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

// IsCodeFile returns true if the file maps to a known language.
func IsCodeFile(path string) bool {
	return DetectLanguage(path) != "Unknown"
}
