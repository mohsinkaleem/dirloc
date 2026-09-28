package output

import (
	"encoding/json"
	"io"

	"github.com/mohsinkaleem/dirloc/types"
)

func renderJSON(w io.Writer, r Report) error {
	out := types.ScanOutput{Summary: r.Summary}
	if !r.Config.NoTopFiles {
		out.TopFiles = r.Files
	}
	if !r.Config.NoTopDirs {
		out.TopDirs = r.Dirs
	}
	if r.Config.ShowLang {
		out.Languages = r.Langs
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
