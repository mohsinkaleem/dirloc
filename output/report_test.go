package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mohsinkaleem/dirloc/types"
)

func sampleReport(format string) Report {
	return Report{
		Summary: types.ScanSummary{TotalFiles: 2, TotalCode: 1500, TotalLines: 1800, Languages: 2, Directories: 1},
		Files: []types.FileResult{
			{Path: "src/a|b.go", Language: "Go", Code: 1200, Total: 1400},
			{Path: "src/c.py", Language: "Python", Code: 300, Total: 400},
		},
		Dirs: []types.DirStats{{Path: "src", Files: 2, Code: 1500, Total: 1800}},
		Langs: []types.LangSummary{
			{Language: "Go", Files: 1, Code: 1200, Total: 1400},
			{Language: "Python", Files: 1, Code: 300, Total: 400},
		},
		Config: types.ScanConfig{Format: format, ShowLang: true, SortBy: "code"},
	}
}

func TestRenderCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleReport("csv")); err != nil {
		t.Fatal(err)
	}
	blocks := strings.Split(strings.TrimSpace(buf.String()), "\n\n")
	if len(blocks) != 3 {
		t.Fatalf("expected 3 CSV blocks, got %d:\n%s", len(blocks), buf.String())
	}

	rows, err := csv.NewReader(strings.NewReader(blocks[2])).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// Raw numbers, no thousands separators or %, and no SUM footer.
	want := []string{"Go", "1", "1200", "0", "0", "1400", "80.0"}
	if len(rows) != 3 || strings.Join(rows[1], ",") != strings.Join(want, ",") {
		t.Errorf("language rows = %v, want second row %v", rows, want)
	}
}

func TestRenderMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleReport("md")); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"## Top 2 Files by Code Lines",
		`| 1 | src/a\|b.go | Go | 1,200 |`,
		"| SUM | 2 | 1,500 | 0 | 0 | 1,800 | 100.0% |",
		"| Go | 1 | 1,200 | 0 | 0 | 1,400 | 80.0% |",
		"| 1 | src" + string(filepath.Separator) + " | 2 | 1,500 | 1,800 | 100.0% |",
		"```mermaid\npie showData title Code lines by language\n    \"Go\" : 1200\n    \"Python\" : 300\n```",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown missing %q:\n%s", want, out)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleReport("json")); err != nil {
		t.Fatal(err)
	}
	var out types.ScanOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.Summary.TotalCode != 1500 || len(out.TopFiles) != 2 || len(out.Languages) != 2 {
		t.Errorf("unexpected JSON output: %+v", out)
	}
}
