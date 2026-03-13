package analyzer

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"sync"

	"github.com/dirloc/dirloc/pkg/types"
)

const defaultMaxBuf = 1024 * 1024 // 1MB line buffer

var lineCountBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

var newlineBytes = []byte{'\n'}

// CountTotalLines efficiently counts only total lines without code/comment classification.
// Uses byte-level newline counting for maximum throughput.
func CountTotalLines(path, lang string) (*types.FileResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return &types.FileResult{Path: path, Language: lang, Error: err.Error()}, err
	}
	defer f.Close()

	bufPtr := lineCountBufPool.Get().(*[]byte)
	defer lineCountBufPool.Put(bufPtr)
	buf := *bufPtr

	total := 0
	trailingNewline := true
	for {
		n, err := f.Read(buf)
		if n > 0 {
			total += bytes.Count(buf[:n], newlineBytes)
			trailingNewline = buf[n-1] == '\n'
		}
		if err != nil {
			break
		}
	}
	if !trailingNewline {
		total++
	}

	return &types.FileResult{
		Path:     path,
		Language: lang,
		Total:    total,
	}, nil
}

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
// Groups related keywords to avoid double-counting overlaps (e.g., "elif" containing "if").
func countBranchKeywords(line string) int {
	count := 0
	lower := strings.ToLower(line)

	// Conditional: elif/elsif take priority over if to avoid double-counting
	if strings.Contains(lower, "elif ") || strings.Contains(lower, "elsif ") {
		count++
	} else if strings.Contains(lower, "if ") || strings.Contains(lower, "if(") || strings.Contains(lower, "unless ") {
		count++
	}

	if strings.Contains(lower, "for ") || strings.Contains(lower, "for(") {
		count++
	}
	if strings.Contains(lower, "while ") || strings.Contains(lower, "while(") {
		count++
	}
	if strings.Contains(lower, "switch ") || strings.Contains(lower, "switch(") {
		count++
	}
	if strings.Contains(lower, "case ") {
		count++
	}
	if strings.Contains(lower, "else ") || strings.Contains(lower, "else{") {
		count++
	}
	if strings.Contains(lower, "catch ") || strings.Contains(lower, "catch(") {
		count++
	}
	if strings.Contains(lower, "except ") || strings.Contains(lower, "except:") {
		count++
	}
	if strings.Contains(lower, "? ") {
		count++
	}

	return count
}
