package parser

import (
	"fmt"
	"go/token"
	"testing"
)

var validFiles = []string{
	"testdata/vldb.bib",
}

func TestParse(t *testing.T) {
	for _, filename := range validFiles {
		_, err := ParseFile(token.NewFileSet(), filename, nil, DeclarationErrors)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", filename, err)
		}
	}
}

func TestParsePreamble(t *testing.T) {
	f, err := ParseFile(token.NewFileSet(), "", "@PREAMBLE { {foo} }", 0)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(f)

}
