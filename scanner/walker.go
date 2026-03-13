package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/dirloc/dirloc/types"
)

// Walk traverses the directory tree starting at root, sending file paths to the returned channel.
// It skips directories and files according to the ignore rules and only sends code files.
func Walk(ctx context.Context, root string, ignore *IgnoreRules, maxFileSize int64) (<-chan string, <-chan string, error) {
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
				if path != root && ignore.ShouldSkipDir(name) {
					return fs.SkipDir
				}
				// Skip symlinked directories to avoid cycles
				if d.Type()&fs.ModeSymlink != 0 {
					return fs.SkipDir
				}
				return nil
			}

			// Skip ignored extensions
			if ignore.ShouldSkipFile(name) {
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
func ProcessFiles(ctx context.Context, paths <-chan string, config types.ScanConfig) <-chan types.FileResult {
	results := make(chan types.FileResult, 256)

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

				// Skip binary files (checked here to avoid double file-open)
				if IsBinary(path) {
					continue
				}

				var result *types.FileResult
				if config.ShowLang || config.ShowComplexity {
					prefixes := GetCommentPrefixes(lang)
					result, _ = CountLines(path, lang, prefixes, config.ShowComplexity)
				} else {
					result, _ = CountTotalLines(path, lang)
				}

				// Make path relative to root for cleaner output
				if rel, err := filepath.Rel(config.RootPath, result.Path); err == nil {
					result.Path = rel
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
