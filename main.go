package main

import (
	_ "embed"

	"github.com/dirloc/dirloc/cmd"
	"github.com/dirloc/dirloc/scanner"
)

//go:embed languages.json
var languagesJSON []byte

func main() {
	scanner.InitLanguages(languagesJSON)
	cmd.Execute()
}
