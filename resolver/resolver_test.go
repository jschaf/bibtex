package resolver

import (
	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/b2/pkg/bibtex"
	"testing"
)

func newAuthor(names ...string) bibtex.Author {
	switch len(names) {
	case 0:
		panic("need at least 1 name")
	case 1:
		return bibtex.Author{
			Last: names[0],
		}
	case 2:
		return bibtex.Author{
			First: names[0],
			Last:  names[1],
		}
	case 3:
		return bibtex.Author{
			First:  names[0],
			Prefix: names[1],
			Last:   names[2],
		}
	case 4:
		return bibtex.Author{
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
		want    bibtex.Author
		wantErr bool
	}{
		{"First von Last", newAuthor("First", "von", "Last"), false},
	}
	for _, tt := range tests {
		t.Run(tt.authors, func(t *testing.T) {
			got, err := ResolveAuthors(tt.authors)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveAuthors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff([]bibtex.Author{tt.want}, got); diff != "" {
				t.Errorf("ResolveAuthors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
