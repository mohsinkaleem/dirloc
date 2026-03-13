package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/dirloc/dirloc/types"
)

const cacheFileName = ".dirlocache"
const cacheVersion = 1

// CacheEntry stores cached line-count results for a single file.
type CacheEntry struct {
	ModTime    int64            `json:"mod_time"`
	Size       int64            `json:"size"`
	Detailed   bool             `json:"detailed"`
	Complexity bool             `json:"complexity"`
	Result     types.FileResult `json:"result"`
}

// Cache provides a file-backed cache of scan results keyed by relative path.
// It is safe for concurrent use from multiple worker goroutines.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	dirty   bool
	path    string
}

type cacheFile struct {
	Version int                    `json:"version"`
	Entries map[string]*CacheEntry `json:"entries"`
}

// LoadCache loads the cache from disk, or returns an empty cache if the file
// does not exist or is corrupt.
func LoadCache(root string) *Cache {
	c := &Cache{
		entries: make(map[string]*CacheEntry),
		path:    filepath.Join(root, cacheFileName),
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		return c
	}

	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil || cf.Version != cacheVersion {
		return c
	}

	c.entries = cf.Entries
	if c.entries == nil {
		c.entries = make(map[string]*CacheEntry)
	}
	return c
}

// Lookup returns a cached result if the file's mtime and size match and the
// cached entry was computed with at least the requested detail level.
func (c *Cache) Lookup(relPath string, modTime, size int64, needDetailed, needComplexity bool) (*types.FileResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[relPath]
	if !ok {
		return nil, false
	}
	if entry.ModTime != modTime || entry.Size != size {
		return nil, false
	}
	if needDetailed && !entry.Detailed {
		return nil, false
	}
	if needComplexity && !entry.Complexity {
		return nil, false
	}
	result := entry.Result
	return &result, true
}

// Store saves a result into the cache.
func (c *Cache) Store(relPath string, modTime, size int64, detailed, complexity bool, result types.FileResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[relPath] = &CacheEntry{
		ModTime:    modTime,
		Size:       size,
		Detailed:   detailed,
		Complexity: complexity,
		Result:     result,
	}
	c.dirty = true
}

// Save writes the cache to disk if it has been modified.
func (c *Cache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.dirty {
		return nil
	}

	cf := cacheFile{
		Version: cacheVersion,
		Entries: c.entries,
	}
	data, err := json.Marshal(cf)
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}
