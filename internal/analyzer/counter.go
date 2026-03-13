package analyzer

import (
	"bufio"
	"os"
	"strings"

	"github.com/dirloc/dirloc/pkg/types"
)

const defaultMaxBuf = 1024 * 1024 // 1MB line buffer

// CountLines counts code, comment, and blank lines in a file.
func CountLines(path, lang string, commentPrefixes []string, countComplexity bool) (*types.FileResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return &types.FileResult{Path: path, Language: lang, Error: err.Error()}, err
	}
	defer f.Close()

	result := &types.FileResult{
		Path:     path,
		Language: lang,
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, defaultMaxBuf), defaultMaxBuf)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			result.Blank++
			continue
		}

		isComment := false
		for _, prefix := range commentPrefixes {
			if strings.HasPrefix(trimmed, prefix) {
				isComment = true
				break
			}
		}

		if isComment {
			result.Comment++
		} else {
			result.Code++
			if countComplexity {
				result.Complexity += countBranchKeywords(trimmed)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		result.Error = err.Error()
	}

	result.Total = result.Code + result.Comment + result.Blank
	return result, nil
}

// countBranchKeywords counts branch/loop keywords on a line.
func countBranchKeywords(line string) int {
	count := 0
	keywords := []string{
		"if ", "if(", "for ", "for(", "while ", "while(",
		"switch ", "switch(", "case ", "else ", "else{",
		"elif ", "elsif ", "catch ", "catch(",
		"except ", "except:", "unless ",
		"? ", // ternary
	}
	lower := strings.ToLower(line)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			count++
		}
	}
	return count
}
