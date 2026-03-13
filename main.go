package main

import (
	_ "embed"

	"github.com/dirloc/dirloc/cmd"
	"github.com/dirloc/dirloc/internal/analyzer"
)

//go:embed languages.json
var languagesJSON []byte

func main() {
	analyzer.InitLanguages(languagesJSON)
	cmd.Execute()
}
