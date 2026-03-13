package types

// FileResult holds line-count results for a single file.
type FileResult struct {
	Path       string `json:"path"`
	Language   string `json:"language"`
	Code       int    `json:"code"`
	Comment    int    `json:"comment"`
	Blank      int    `json:"blank"`
	Total      int    `json:"total"`
	Complexity int    `json:"complexity,omitempty"`
	Error      string `json:"error,omitempty"`
}

// DirStats holds aggregated stats for a directory.
type DirStats struct {
	Path    string `json:"path"`
	Files   int    `json:"files"`
	Code    int    `json:"code"`
	Comment int    `json:"comment"`
	Blank   int    `json:"blank"`
	Total   int    `json:"total"`
}

// LangSummary holds aggregated stats for a language.
type LangSummary struct {
	Language string `json:"language"`
	Files    int    `json:"files"`
	Code     int    `json:"code"`
	Comment  int    `json:"comment"`
	Blank    int    `json:"blank"`
	Total    int    `json:"total"`
}

// ScanSummary holds the overall scan summary.
type ScanSummary struct {
	TotalFiles   int `json:"total_files"`
	TotalCode    int `json:"total_code"`
	TotalComment int `json:"total_comment"`
	TotalBlank   int `json:"total_blank"`
	TotalLines   int `json:"total_lines"`
	Languages    int `json:"languages"`
	Directories  int `json:"directories"`
	Errors       int `json:"errors"`
}

// ScanConfig holds all configuration from CLI flags.
type ScanConfig struct {
	RootPath       string
	ExcludeDirs    []string
	ExcludeExts    []string
	ExcludeFiles   []string
	IncludeExts    []string
	IncludeLangs   []string
	Workers        int
	TopK           int
	ShowLang       bool
	ShowComplexity bool
	OutputJSON     bool
	OutputMD       bool
	NoTopFiles     bool
	NoTopDirs      bool
	SortBy         string
	MaxFileSize    int64
	UseGitignore   bool
	UseCache       bool
	NoProgress     bool
	MaxDepth       int
}

// ScanOutput wraps all output data for JSON/MD rendering.
type ScanOutput struct {
	Summary   ScanSummary   `json:"summary"`
	TopFiles  []FileResult  `json:"top_files,omitempty"`
	TopDirs   []DirStats    `json:"top_dirs,omitempty"`
	Languages []LangSummary `json:"languages,omitempty"`
}
