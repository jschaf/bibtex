package bibtex

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/asts"
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
				  title={Learning from satisfying assignments under {Continuous} distributions},
				  author={Canonne {Foo}, Clement L and De, Anindya and Servedio, Rocco A},
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
					"title":        asts.Text("Learning from satisfying assignments under Continuous distributions"),
					"author":       newAuthors(newAuthor("Clement L", "Canonne Foo"), newAuthor("Anindya", "De"), newAuthor("Rocco A", "Servedio")),
					"year":         asts.Text("2020"),
				},
			},
		},
		{
			name: "book linear algebra",
			src: `
				@book{greub2012linear,
				  title={Linear algebra},
				  author={Greub, Werner H},
				  volume={23},
				  year={2012},
				  publisher={Springer Science \& Business Media}
				}`,
			want: Entry{
				Type: EntryBook,
				Key:  "greub2012linear",
				Tags: map[Field]ast.Expr{
					"title":     asts.Text("Linear algebra"),
					"author":    newAuthors(newAuthor("Werner H", "Greub")),
					"publisher": asts.Text("Springer Science & Business Media"),
					"year":      asts.Text("2012"),
					"volume":    asts.Text("23"),
				},
			},
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
			src:  `@article{cite_key, url = "https://example.com/foo--bar/~baz/#" }`,
			want: Entry{Type: EntryArticle, Key: "cite_key", Tags: map[Field]ast.Expr{"url": asts.Text("https://example.com/foo--bar/~baz/#")}},
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

func ExampleNew_renderToString() {
	input := `
    @book{greub2012linear,
      title={Linear algebra},
      author={Greub, {WERNER} H},
      volume={23},
      year={2012},
      publisher={Springer Science \& Business Media}
    }

    @inproceedings{francese2015model,
      title={Model-driven development for multi-platform mobile applications},
      author={Francese, Rita and Risi, Michele and Scanniello, Giuseppe and Tortora, Genoveffa},
      booktitle={Product-Focused Software Process Improvement: 16th International Conference, PROFES 2015, Bolzano, Italy, December 2-4, 2015, Proceedings 16},
      pages={61--67},
      year={2015},
      organization={Springer}
    }`

	bib := New(
		WithResolvers(
			// NewAuthorResolver creates a resolver for the "author" field that parses
			// author names into an ast.Authors node.
			NewAuthorResolver("author"),
			// SimplifyEscapedTextResolver replaces ast.TextEscaped nodes with a plain
			// ast.Text containing the value that was escaped. Meaning, `\&` is converted to
			// `&`.
			ResolverFunc(SimplifyEscapedTextResolver),
			// RenderParsedTextResolver replaces ast.ParsedText with a simplified rendering
			// of ast.Text.
			NewRenderParsedTextResolver(),
		),
	)

	file, err := bib.Parse(strings.NewReader(input))
	if err != nil {
		panic(err.Error())
	}
	entries, err := bib.Resolve(file)
	if err != nil {
		panic(err.Error())
	}

	// Use intermediate type since tag output order is not deterministic.
	// Go maps are unordered.
	type TagOutput struct {
		Field string
		Value string
	}
	type EntryOutput struct {
		Type string
		Key  string
		Tags []TagOutput
	}
	entryOutputs := make([]EntryOutput, 0, len(entries))
	for _, entry := range entries {
		tags := make([]TagOutput, 0, len(entry.Tags))
		for field, expr := range entry.Tags {
			switch expr := expr.(type) {
			case ast.Authors:
				sb := strings.Builder{}
				if len(expr) > 0 {
					for i, author := range expr {
						if i > 0 {
							sb.WriteString("\n")
						}
						first := author.First.(*ast.Text).Value
						prefix := author.Prefix.(*ast.Text).Value
						last := author.Last.(*ast.Text).Value
						suffix := author.Suffix.(*ast.Text).Value
						name := fmt.Sprintf("%s %s %s %s", first, prefix, last, suffix)
						name = strings.TrimSpace(name)
						name = strings.Join(strings.Fields(name), " ") // remove consecutive spaces
						sb.WriteString(field)
						sb.WriteString(": ")
						sb.WriteString(name)
					}
					tags = append(tags, TagOutput{Field: field, Value: sb.String()})
				}
			case *ast.Text:
				tags = append(tags, TagOutput{Field: field, Value: fmt.Sprintf("%s: %s", field, expr.Value)})
			default:
				tags = append(tags, TagOutput{Field: field, Value: fmt.Sprintf("%s: %T", field, expr)})
			}
		}
		sort.Slice(tags, func(i, j int) bool {
			return tags[i].Field < tags[j].Field
		})
		entryOutputs = append(entryOutputs, EntryOutput{
			Type: entry.Type,
			Key:  entry.Key,
			Tags: tags,
		})
	}

	for _, out := range entryOutputs {
		fmt.Printf("type: %s\n", out.Type)
		fmt.Printf("key: %s\n", out.Key)
		for _, tag := range out.Tags {
			fmt.Println(tag.Value)
		}
		fmt.Println()
	}

	// Output:
	// type: book
	// key: greub2012linear
	// author: WERNER H Greub
	// publisher: Springer Science & Business Media
	// title: Linear algebra
	// volume: 23
	// year: 2012
	//
	// type: inproceedings
	// key: francese2015model
	// author: Rita Francese
	// author: Michele Risi
	// author: Giuseppe Scanniello
	// author: Genoveffa Tortora
	// booktitle: Product-Focused Software Process Improvement: 16th International Conference, PROFES 2015, Bolzano, Italy, December 2-4, 2015, Proceedings 16
	// organization: Springer
	// pages: 61--67
	// title: Model-driven development for multi-platform mobile applications
	// year: 2015
	//
}
