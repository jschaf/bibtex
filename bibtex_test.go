package bibtex

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/asts"
	"strings"
	"testing"
)

func TestNew_resolve(t *testing.T) {
	cmpOpts := cmp.Options{
		cmpopts.IgnoreFields(ast.Text{}, "ValuePos"),
	}
	tests := []struct {
		name string
		src  string
		want Entry
	}{
		{
			name: "inproceedings",
			src: `
				@inproceedings{canonne2020learning,
				  title={Learning from satisfying assignments under continuous distributions},
				  author={Canonne, Clement L and De, Anindya and Servedio, Rocco A},
				  booktitle={Proceedings of the Fourteenth Annual ACM-SIAM Symposium on Discrete Algorithms},
				  pages={82--101},
				  year={2020},
				  organization={SIAM}
			  }`,
			want: Entry{
				Type: EntryInProceedings,
				Key:  "canonne2020learning",
				Tags: map[Field]ast.Expr{
					"booktitle":    asts.Text("Proceedings of the Fourteenth Annual ACM-SIAM Symposium on Discrete Algorithms"),
					"organization": asts.Text("SIAM"),
					"pages":        asts.Text("82--101"),
					"title":        asts.Text("Learning from satisfying assignments under continuous distributions"),
					"author":       newAuthors(newAuthor("Clement L", "Canonne"), newAuthor("Anindya", "De"), newAuthor("Rocco A", "Servedio")),
					"year":         asts.Text("2020"),
				}},
		},
		{
			name: "book with only title",
			src:  `@book{citekey, title={Foo \& Bar \$1} }`,
			want: Entry{Type: EntryBook, Key: "citekey", Tags: map[Field]ast.Expr{"title": asts.Text("Foo & Bar $1")}},
		},
		{
			name: "book with math title",
			src:  `@article{citekey, title={formula $e=mc^2$} }`,
			want: Entry{Type: EntryArticle, Key: "citekey", Tags: map[Field]ast.Expr{"title": asts.Text("formula $e=mc^2$")}},
		},
		{
			name: "book with url",
			src:  `@book{citekey, url={\url{www.example.com}} }`,
			want: Entry{Type: EntryBook, Key: "citekey", Tags: map[Field]ast.Expr{"url": asts.Text("www.example.com")}},
		},
		{
			name: "article with url",
			src:  `@article{cite_key, url = "http://example.com/foo--bar/~baz/#" }`,
			want: Entry{Type: EntryArticle, Key: "cite_key", Tags: map[Field]ast.Expr{"url": asts.Text("http://example.com/foo--bar/~baz/#")}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bib := New(
				WithResolvers(
					NewAuthorResolver("author"),
					ResolverFunc(SimplifyEscapedTextResolver),
					NewRenderParsedTextResolver(),
				))
			file, err := bib.Parse(strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			entries, err := bib.Resolve(file)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected exactly 1 entry, got %d entries", len(entries))
			}

			if diff := cmp.Diff(tt.want, entries[0], cmpOpts); diff != "" {
				t.Errorf("Read() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
