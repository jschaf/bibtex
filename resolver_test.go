package bibtex

import (
	"github.com/jschaf/bibtex/asts"
	gotok "go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/parser"
)

func TestResolveAuthors_single(t *testing.T) {
	tests := []struct {
		authors string
		want    Author
	}{
		{"Last", newAuthor("Last")},
		{"First Last", newAuthor("First", "Last")},
		{"First last", newAuthor("First", "last")},
		{"last", newAuthor("last")},
		{"First von Last", newAuthor("First", "von", "Last")},
		// {"First aa Von bb Last", author("First", "aa Von bb", "Last")},
		{"von Beethoven, Ludwig", newAuthor("Ludwig", "von", "Beethoven")},
		{"{von Beethoven}, Ludwig", newAuthor("Ludwig", "von Beethoven")},
		{"Jean-Paul Sartre", newAuthor("Jean-Paul", "Sartre")},
		{"First von Last", newAuthor("First", "von", "Last")},
		{"Charles Louis Xavier Joseph de la Vallee Poussin",
			newAuthor("Charles Louis Xavier Joseph", "de la", "Vallee Poussin"),
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
		wantErr bool
	}{
		{"Last and Last2", []Author{newAuthor("Last"), newAuthor("Last2")}, false},
		// double and should not cause a crash, instead it stops at the first empty and returns and err
		{"Last3 and and Last4", []Author{newAuthor("Last3")}, true},
		{"F1 L1 and F2 L2", []Author{newAuthor("F1", "L1"), newAuthor("F2", "L2")}, false},
		{"F1 L1 and L2, F2", []Author{newAuthor("F1", "L1"), newAuthor("F2", "L2")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.authors, func(t *testing.T) {
			a, err := parser.ParseExpr("{" + tt.authors + "}")
			if err != nil {
				t.Fatal(err)
			}
			got, err := ResolveAuthors(a.(*ast.ParsedText))
			if err != nil && !tt.wantErr {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ResolveAuthors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type testASTEntry struct {
	Type EntryType
	Key  CiteKey
	Tags map[Field]string
}

// cmpASTEntry is a functional option for cmp.Diff to normalize ASTEntry for
// easier comparison
func cmpASTEntry() cmp.Option {
	return cmp.Transformer("cmpASTEntry", func(x ASTEntry) testASTEntry {
		t := testASTEntry{
			Type: x.Type,
			Key:  x.Key,
			Tags: make(map[Field]string, len(x.Tags)),
		}
		for field, tag := range x.Tags {
			t.Tags[field] = asts.ExprString(tag)
		}
		return t
	})
}

func TestResolveFile(t *testing.T) {
	tests := []struct {
		src  string
		want []ASTEntry
	}{
		{"@article{key, author = {Foo Bar}}", []ASTEntry{
			{
				Type: EntryArticle, Key: "key",
				Tags: map[Field]ast.Expr{
					"author": asts.BraceText(0, "Foo", asts.Space(), "Bar"),
				},
			},
		}},
		{"@article{abc, author = {Moir, Mark and Scherer,III, William N.}}", []ASTEntry{
			{
				Type: EntryArticle, Key: "abc",
				Tags: map[Field]ast.Expr{
					"author": asts.BraceText(0,
						"Moir", asts.Comma(), asts.Space(), "Mark", asts.Space(), "and",
						asts.Space(), "Scherer", asts.Comma(), "III", asts.Comma(), asts.Space(),
						"William", asts.Space(), "N.",
					),
				},
			},
		}},
		{"@article{ rfc1812, author = {F. {Baker, ed.}}}", []ASTEntry{
			{
				Type: EntryArticle, Key: "rfc1812",
				Tags: map[Field]ast.Expr{
					"author": asts.BraceText(0, "F.", asts.Space(), asts.BraceText(1, "Baker", asts.Comma(), asts.Space(), "ed.")),
				},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got, err := ResolveFile(gotok.NewFileSet(), "", tt.src)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.want, got, cmpASTEntry()); diff != "" {
				t.Errorf("ResolveFile() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
