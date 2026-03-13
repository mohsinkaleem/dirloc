package scanner

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/dirloc/dirloc/types"
)

const defaultMaxBuf = 1024 * 1024 // 1MB line buffer

var lineCountBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

var scannerBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 0, defaultMaxBuf)
		return &buf
	},
}

var newlineBytes = []byte{'\n'}

// CountTotalLines efficiently counts only total lines without code/comment classification.
// Uses byte-level newline counting for maximum throughput.
// It also performs binary detection on the first chunk read.
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
	firstChunk := true
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if firstChunk {
				firstChunk = false
				// Binary detection: check for null bytes in first chunk
				for _, b := range buf[:n] {
					if b == 0 {
						return nil, nil // signal: binary file, skip
					}
				}
			}
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

// CountLines counts code, comment, blank, and (optionally) block-comment lines in a file.
// It also performs binary detection on the first 512 bytes.
func CountLines(path, lang string, commentPrefixes []string, blockStart, blockEnd string, countComplexity bool) (*types.FileResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return &types.FileResult{Path: path, Language: lang, Error: err.Error()}, err
	}
	defer f.Close()

	// Binary detection: read first 512 bytes
	var peekBuf [512]byte
	n, _ := io.ReadFull(f, peekBuf[:])
	if n > 0 {
		for _, b := range peekBuf[:n] {
			if b == 0 {
				return nil, nil // binary file, skip
			}
		}
	}
	// Seek back to start for full scan
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return &types.FileResult{Path: path, Language: lang, Error: err.Error()}, err
	}

	result := &types.FileResult{
		Path:     path,
		Language: lang,
	}

	bufPtr := scannerBufPool.Get().(*[]byte)
	s := bufio.NewScanner(f)
	s.Buffer(*bufPtr, defaultMaxBuf)

	inBlock := false
	hasBlockComments := blockStart != "" && blockEnd != ""

	for s.Scan() {
		line := s.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			result.Blank++
			continue
		}

		// Block comment handling
		if hasBlockComments {
			if inBlock {
				result.Comment++
				if strings.Contains(trimmed, blockEnd) {
					inBlock = false
				}
				continue
			}
			if strings.Contains(trimmed, blockStart) {
				result.Comment++
				if !strings.Contains(trimmed, blockEnd) {
					inBlock = true
				}
				continue
			}
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

	if err := s.Err(); err != nil {
		result.Error = err.Error()
	}

	*bufPtr = (*bufPtr)[:0] // reset length, keep capacity
	scannerBufPool.Put(bufPtr)

	result.Total = result.Code + result.Comment + result.Blank
	return result, nil
}

// countBranchKeywords counts branch/loop keywords on a line.
// Uses case-insensitive contains to avoid per-line string allocation.
func countBranchKeywords(line string) int {
	count := 0

	// Conditional: elif/elsif take priority over if to avoid double-counting
	if containsCI(line, "elif ") || containsCI(line, "elsif ") {
		count++
	} else if containsCI(line, "if ") || containsCI(line, "if(") || containsCI(line, "unless ") {
		count++
	}

	if containsCI(line, "for ") || containsCI(line, "for(") {
		count++
	}
	if containsCI(line, "while ") || containsCI(line, "while(") {
		count++
	}
	if containsCI(line, "switch ") || containsCI(line, "switch(") {
		count++
	}
	if containsCI(line, "case ") {
		count++
	}
	if containsCI(line, "else ") || containsCI(line, "else{") {
		count++
	}
	if containsCI(line, "catch ") || containsCI(line, "catch(") {
		count++
	}
	if containsCI(line, "except ") || containsCI(line, "except:") {
		count++
	}
	if strings.Contains(line, "? ") {
		count++
	}

	return count
}

// containsCI is a case-insensitive strings.Contains that avoids allocation.
func containsCI(s, substr string) bool {
	n := len(substr)
	if n == 0 {
		return true
	}
	if n > len(s) {
		return false
	}
	for i := 0; i <= len(s)-n; i++ {
		match := true
		for j := 0; j < n; j++ {
			sc := s[i+j]
			pc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 'a' - 'A'
			}
			if sc != pc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
