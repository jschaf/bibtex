package bibtex

import (
	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/parser"
	gotok "go/token"
	"testing"
)

func author(names ...string) Author {
	switch len(names) {
	case 0:
		panic("need at least 1 name")
	case 1:
		return Author{
			Last: names[0],
		}
	case 2:
		return Author{
			First: names[0],
			Last:  names[1],
		}
	case 3:
		return Author{
			First:  names[0],
			Prefix: names[1],
			Last:   names[2],
		}
	case 4:
		return Author{
			First:  names[0],
			Prefix: names[1],
			Last:   names[2],
			Suffix: names[3],
		}
	default:
		panic("too many names")
	}
}

func TestResolveAuthors_single(t *testing.T) {
	tests := []struct {
		authors string
		want    Author
	}{
		{"Last", author("Last")},
		{"First Last", author("First", "Last")},
		{"First last", author("First", "last")},
		{"last", author("last")},
		{"First von Last", author("First", "von", "Last")},
		// {"First aa Von bb Last", author("First", "aa Von bb", "Last")},
		{"von Beethoven, Ludwig", author("Ludwig", "von", "Beethoven")},
		{"{von Beethoven}, Ludwig", author("Ludwig", "von Beethoven")},
		{"Jean-Paul Sartre", author("Jean-Paul", "Sartre")},
		{"First von Last", author("First", "von", "Last")},
		{"Charles Louis Xavier Joseph de la Vallee Poussin",
			author("Charles Louis Xavier Joseph", "de la", "Vallee Poussin"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.authors, func(t *testing.T) {
			a, err := parser.ParseExpr("{" + tt.authors + "}")
			if err != nil {
				t.Fatal(err)
			}
			got, _ := ResolveAuthors(a.(*ast.ParsedText))
			if diff := cmp.Diff([]Author{tt.want}, got); diff != "" {
				t.Errorf("ResolveAuthors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveAuthors_multiple(t *testing.T) {
	tests := []struct {
		authors string
		want    []Author
	}{
		{"Last and Last2", []Author{author("Last"), author("Last2")}},
		{"F1 L1 and F2 L2", []Author{author("F1", "L1"), author("F2", "L2")}},
		{"F1 L1 and L2, F2", []Author{author("F1", "L1"), author("F2", "L2")}},
	}
	for _, tt := range tests {
		t.Run(tt.authors, func(t *testing.T) {
			a, err := parser.ParseExpr("{" + tt.authors + "}")
			if err != nil {
				t.Fatal(err)
			}
			got, _ := ResolveAuthors(a.(*ast.ParsedText))
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ResolveAuthors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveFile(t *testing.T) {
	tests := []struct {
		src  string
		want []Entry
	}{
		{"@article{key, author = {Foo Bar}}", []Entry{
			{Type: EntryArticle, Key: "key",
				Tags:   make(map[Field]string),
				Author: []Author{author("Foo", "Bar")}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got, err := ResolveFile(gotok.NewFileSet(), "", tt.src)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ResolveFile() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
