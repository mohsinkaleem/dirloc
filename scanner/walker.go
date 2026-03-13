package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dirloc/dirloc/types"
)

// Walk traverses the directory tree starting at root, sending file paths to the returned channel.
// It skips directories and files according to the ignore rules and only sends code files.
// If gitMatcher is non-nil, .gitignore files are loaded and honoured.
// If progress is non-nil, it is incremented for every file emitted.
// maxDepth limits traversal depth (0 = unlimited).
func Walk(ctx context.Context, root string, ignore *IgnoreRules, maxFileSize int64, gitMatcher *GitIgnoreMatcher, progress *Progress, maxDepth int) (<-chan string, <-chan string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot access %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("%s is not a directory", root)
	}

	paths := make(chan string, 256)
	warnings := make(chan string, 64)

	go func() {
		defer close(paths)
		defer close(warnings)

		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				warnings <- fmt.Sprintf("warning: %s: %v", path, err)
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			name := d.Name()

			if d.IsDir() {
				// Enforce --depth limit
				if maxDepth > 0 && path != root {
					rel, _ := filepath.Rel(root, path)
					depth := strings.Count(rel, string(filepath.Separator)) + 1
					if depth > maxDepth {
						return fs.SkipDir
					}
				}

				if path != root && ignore.ShouldSkipDir(name) {
					return fs.SkipDir
				}
				// Skip symlinked directories to avoid cycles
				if d.Type()&fs.ModeSymlink != 0 {
					return fs.SkipDir
				}
				// Load .gitignore for this directory if enabled
				if gitMatcher != nil {
					gitMatcher.LoadDir(path)
				}
				// Check gitignore rules on the directory itself
				if gitMatcher != nil && path != root && gitMatcher.ShouldIgnore(path, true) {
					return fs.SkipDir
				}
				return nil
			}

			// Skip ignored files by name
			if ignore.ShouldSkipFile(name) {
				return nil
			}

			// Skip gitignored files
			if gitMatcher != nil && gitMatcher.ShouldIgnore(path, false) {
				return nil
			}

			// Skip non-code files
			if !IsCodeFile(path) {
				return nil
			}

			// Check file size
			if maxFileSize > 0 {
				info, err := d.Info()
				if err != nil {
					warnings <- fmt.Sprintf("warning: cannot stat %s: %v", path, err)
					return nil
				}
				if info.Size() > maxFileSize {
					warnings <- fmt.Sprintf("warning: skipping %s (%.1f MB exceeds max)", path, float64(info.Size())/(1024*1024))
					return nil
				}
			}

			progress.Inc()

			select {
			case paths <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}()

	return paths, warnings, nil
}

// ProcessFiles spawns worker goroutines to analyze files from the paths channel.
// If cache is non-nil, workers check it before reading files and store results.
func ProcessFiles(ctx context.Context, paths <-chan string, config types.ScanConfig, cache *Cache) <-chan types.FileResult {
	results := make(chan types.FileResult, 256)

	needDetailed := config.ShowLang || config.ShowComplexity
	needComplexity := config.ShowComplexity

	var wg sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range paths {
				select {
				case <-ctx.Done():
					return
				default:
				}

				lang := DetectLanguage(path)

				// Make path relative to root for cleaner output
				relPath := path
				if rel, err := filepath.Rel(config.RootPath, path); err == nil {
					relPath = rel
				}

				// Stat once and reuse for cache lookup/store
				var fileInfo os.FileInfo
				if cache != nil {
					fi, err := os.Stat(path)
					if err == nil {
						fileInfo = fi
						if cached, ok := cache.Lookup(relPath, fi.ModTime().UnixNano(), fi.Size(), needDetailed, needComplexity); ok {
							select {
							case results <- *cached:
							case <-ctx.Done():
								return
							}
							continue
						}
					}
				}

				var result *types.FileResult
				if needDetailed {
					prefixes := GetCommentPrefixes(lang)
					blockStart, blockEnd := GetBlockCommentDelimiters(lang)
					result, _ = CountLines(path, lang, prefixes, blockStart, blockEnd, needComplexity)
				} else {
					result, _ = CountTotalLines(path, lang)
				}

				// nil result means binary file detected inside count function
				if result == nil {
					continue
				}

				result.Path = relPath

				// Store in cache (reuse fileInfo if available)
				if cache != nil {
					if fileInfo == nil {
						fileInfo, _ = os.Stat(path)
					}
					if fileInfo != nil {
						cache.Store(relPath, fileInfo.ModTime().UnixNano(), fileInfo.Size(), needDetailed, needComplexity, *result)
					}
				}

				select {
				case results <- *result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}
