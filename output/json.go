package output

import (
	"encoding/json"
	"os"
	"time"

	"github.com/dirloc/dirloc/types"
)

// RenderJSON outputs scan results as JSON.
func RenderJSON(summary types.ScanSummary, topFiles []types.FileResult, topDirs []types.DirStats, langSummaries []types.LangSummary, config types.ScanConfig, elapsed time.Duration) error {
	out := types.ScanOutput{
		Summary: summary,
	}

	if !config.NoTopFiles {
		out.TopFiles = topFiles
	}
	if !config.NoTopDirs {
		out.TopDirs = topDirs
	}
	if config.ShowLang {
		out.Languages = langSummaries
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
