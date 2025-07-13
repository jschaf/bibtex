package bibtex

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/parser"
)

func TestResolveAuthors_single(t *testing.T) {
	tests := []struct {
		authors string
		want    *ast.Author
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
		{"First von Last", newAuthor("First", "von", "Last")},
		{"Beno{\\^i}t de Meg\\`eve", newAuthor("Benoît", "de", "Megève")},
		{"Fran{\\c{c}}oise Chollet", newAuthor("Françoise", "Chollet")},
		{"Fran{\\cc}oise Chollet", newAuthor("Françoise", "Chollet")},
		{"Fran{\\c c}oise Chollet", newAuthor("Françoise", "Chollet")},
		{
			"Charles Louis Xavier Joseph de la Vallee Poussin",
			newAuthor("Charles Louis Xavier Joseph", "de la", "Vallee Poussin"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.authors, func(t *testing.T) {
			a, err := parser.ParseExpr("{" + tt.authors + "}")
			if err != nil {
				t.Fatal(err)
			}
			got, _ := ExtractAuthors(a.(*ast.ParsedText))
			if diff := cmp.Diff(newAuthors(tt.want), got); diff != "" {
				t.Errorf("ExtractAuthors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveAuthors_multiple(t *testing.T) {
	tests := []struct {
		authors string
		want    ast.Authors
		wantErr bool
	}{
		{"Last and Last2", newAuthors(newAuthor("Last"), newAuthor("Last2")), false},
		{"Last3 and and Last4", nil, true},
		{"F1 L1 and F2 L2", newAuthors(newAuthor("F1", "L1"), newAuthor("F2", "L2")), false},
		{"F1 L1 and L2, F2", newAuthors(newAuthor("F1", "L1"), newAuthor("F2", "L2")), false},
	}
	for _, tt := range tests {
		t.Run(tt.authors, func(t *testing.T) {
			a, err := parser.ParseExpr("{" + tt.authors + "}")
			if err != nil {
				t.Fatal(err)
			}
			got, err := ExtractAuthors(a.(*ast.ParsedText))
			if err != nil && !tt.wantErr {
				t.Fatal(err)
			} else if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ExtractAuthors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
