// Package resolver transforms a Bibtex AST into complete bibtex entries,
// resolving cross references, parsing authors and editors, and normalizing
// page numbers.
package resolver

import (
	"github.com/jschaf/b2/pkg/bibtex"
	"strings"
)

type resolver struct {
}

type authorParser struct {
}

func ResolveAuthors(s string) ([]bibtex.Author, error) {
	commas := strings.Count(s, ",")
	switch commas {
	case 0:

	}
	return nil, nil
}
