package bibtex

import (
	"github.com/google/go-cmp/cmp"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	tests := []struct {
		src  string
		want Entry
	}{
		{`
				@inproceedings{canonne2020learning,
				  title={Learning from satisfying assignments under continuous distributions},
				  author={Canonne, Clement L and De, Anindya and Servedio, Rocco A},
				  booktitle={Proceedings of the Fourteenth Annual ACM-SIAM Symposium on Discrete Algorithms},
				  pages={82--101},
				  year={2020},
				  organization={SIAM}
			  }
	  `,
			Entry{
				Type:   EntryInProceedings,
				Key:    "canonne2020learning",
				Author: []Author{newAuthor("Clement L", "Canonne"), newAuthor("Anindya", "De"), newAuthor("Rocco A", "Servedio")},
				Tags: map[Field]string{
					"booktitle":    "Proceedings of the Fourteenth Annual ACM-SIAM Symposium on Discrete Algorithms",
					"organization": "SIAM",
					"pages":        "82--101",
					"title":        "Learning from satisfying assignments under continuous distributions",
					"year":         "2020",
				}},
		},
		{
			`@book{citekey, title={Foo \& Bar \$1} }`,
			Entry{Type: EntryBook, Key: "citekey", Tags: map[Field]string{"title": "Foo & Bar $1"}},
		},
		{
			`@article{citekey, title={formula $e=mc^2$} }`,
			Entry{Type: EntryArticle, Key: "citekey", Tags: map[Field]string{"title": "formula $e=mc^2$"}},
		},
		{
			`@book{citekey, url={\url{www.example.com}} }`,
			Entry{Type: EntryBook, Key: "citekey", Tags: map[Field]string{"url": "www.example.com"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			entries, err := Read(strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected exactly 1 entry, got %d entries", len(entries))
			}

			if diff := cmp.Diff(tt.want, entries[0]); diff != "" {
				t.Errorf("Read() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
