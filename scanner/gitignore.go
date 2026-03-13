package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// gitignorePattern represents one parsed line from a .gitignore file.
type gitignorePattern struct {
	pattern  string
	negated  bool
	dirOnly  bool
	anchored bool // pattern is relative to the .gitignore location
}

// GitIgnoreMatcher accumulates .gitignore rules discovered during a walk and
// tests paths against them.
type GitIgnoreMatcher struct {
	rules []gitignoreRuleSet
}

type gitignoreRuleSet struct {
	baseDir  string
	patterns []gitignorePattern
}

// NewGitIgnoreMatcher creates an empty matcher.
func NewGitIgnoreMatcher() *GitIgnoreMatcher {
	return &GitIgnoreMatcher{}
}

// LoadDir checks for a .gitignore in dir and, if present, parses it.
func (gm *GitIgnoreMatcher) LoadDir(dir string) {
	path := filepath.Join(dir, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var patterns []gitignorePattern
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimRight(s.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		p := gitignorePattern{}

		if strings.HasPrefix(line, "!") {
			p.negated = true
			line = line[1:]
		}

		if strings.HasPrefix(line, "\\#") || strings.HasPrefix(line, "\\!") {
			line = line[1:]
		}

		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}

		// A leading / or a / in the middle anchors the pattern to baseDir.
		if strings.HasPrefix(line, "/") {
			p.anchored = true
			line = line[1:]
		} else if strings.Contains(line, "/") {
			p.anchored = true
		}

		p.pattern = line
		patterns = append(patterns, p)
	}

	if len(patterns) > 0 {
		gm.rules = append(gm.rules, gitignoreRuleSet{baseDir: dir, patterns: patterns})
	}
}

// ShouldIgnore returns true if path should be ignored according to loaded
// .gitignore files. isDir should be true when path is a directory.
func (gm *GitIgnoreMatcher) ShouldIgnore(path string, isDir bool) bool {
	if gm == nil {
		return false
	}
	ignored := false
	for _, rs := range gm.rules {
		rel, err := filepath.Rel(rs.baseDir, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		name := filepath.Base(path)

		for _, p := range rs.patterns {
			if p.dirOnly && !isDir {
				continue
			}

			var matched bool
			if p.anchored {
				matched = matchGitPattern(p.pattern, rel)
			} else {
				matched = matchGitPattern(p.pattern, name)
				if !matched {
					matched = matchGitPattern(p.pattern, rel)
				}
			}

			if matched {
				ignored = !p.negated
			}
		}
	}
	return ignored
}

// matchGitPattern matches a gitignore pattern against a name or relative path.
func matchGitPattern(pattern, name string) bool {
	if strings.Contains(pattern, "**") {
		return matchDoublestar(pattern, name)
	}
	if strings.Contains(pattern, "/") {
		// Match each component
		return matchPathPattern(pattern, name)
	}
	matched, _ := filepath.Match(pattern, name)
	return matched
}

// matchPathPattern matches a pattern containing / against a relative path.
func matchPathPattern(pattern, path string) bool {
	pp := strings.Split(pattern, "/")
	tp := strings.Split(path, string(filepath.Separator))

	if len(tp) < len(pp) {
		return false
	}

	// Try matching at each possible starting position.
	for start := 0; start <= len(tp)-len(pp); start++ {
		ok := true
		for i, p := range pp {
			if matched, _ := filepath.Match(p, tp[start+i]); !matched {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// matchDoublestar handles patterns containing **.
func matchDoublestar(pattern, path string) bool {
	parts := strings.SplitN(pattern, "**", 2)
	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	if prefix == "" && suffix == "" {
		return true
	}

	if prefix == "" {
		// **/suffix – match suffix at any depth.
		pathParts := strings.Split(path, string(filepath.Separator))
		for i := range pathParts {
			sub := strings.Join(pathParts[i:], string(filepath.Separator))
			if matched, _ := filepath.Match(suffix, sub); matched {
				return true
			}
			// Also try matching suffix as a path pattern.
			if strings.Contains(suffix, "/") && matchPathPattern(suffix, sub) {
				return true
			}
		}
		return false
	}

	if suffix == "" {
		return strings.HasPrefix(path, prefix+string(filepath.Separator)) || path == prefix
	}

	// prefix/**/suffix
	if !strings.HasPrefix(path, prefix+string(filepath.Separator)) {
		return false
	}
	rest := path[len(prefix)+1:]
	return matchDoublestar("**/"+suffix, rest)
}
