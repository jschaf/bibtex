package bibtex

import (
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/asts"
)

// newAuthor creates a new author using the number of strings to infer
// the name structure as follows:
//
//	1 strings: Last
//	2 strings: First, Last
//	3 strings: First, Prefix, Last
//	4 strings: First, Prefix, Last, Suffix
func newAuthor(names ...string) *ast.Author {
	switch len(names) {
	case 0:
		panic("need at least 1 name")
	case 1:
		return &ast.Author{
			First:  asts.Text(""),
			Prefix: asts.Text(""),
			Last:   asts.Text(names[0]),
			Suffix: asts.Text(""),
		}
	case 2:
		return &ast.Author{
			First:  asts.Text(names[0]),
			Prefix: asts.Text(""),
			Last:   asts.Text(names[1]),
			Suffix: asts.Text(""),
		}
	case 3:
		return &ast.Author{
			First:  asts.Text(names[0]),
			Prefix: asts.Text(names[1]),
			Last:   asts.Text(names[2]),
			Suffix: asts.Text(""),
		}
	case 4:
		return &ast.Author{
			First:  asts.Text(names[0]),
			Prefix: asts.Text(names[1]),
			Last:   asts.Text(names[2]),
			Suffix: asts.Text(names[3]),
		}
	default:
		panic("too many names")
	}
}

func newAuthors(auths ...*ast.Author) ast.Authors {
	return auths
}
