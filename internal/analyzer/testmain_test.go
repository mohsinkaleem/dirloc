package analyzer

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	data, err := os.ReadFile("../../languages.json")
	if err != nil {
		panic("cannot load languages.json for tests: " + err.Error())
	}
	InitLanguages(data)
	os.Exit(m.Run())
}
