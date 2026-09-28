package main

import (
	_ "embed"

	"github.com/mohsinkaleem/dirloc/cmd"
	"github.com/mohsinkaleem/dirloc/scanner"
)

//go:embed languages.json
var languagesJSON []byte

func main() {
	scanner.InitLanguages(languagesJSON)
	cmd.Execute()
}
