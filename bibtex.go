package bibtex

import (
	"github.com/jschaf/b2/pkg/bibtex/scanner"
	gotok "go/token"
	"io"
)

type CiteKey = string

type EntryType = string

const (
	EntryArticle EntryType = "article"
)

type Field = string

const (
	FieldAddress   = "address"
	FieldAuthor    = "author"
	FieldEditor    = "editor"
	FieldBookTitle = "booktitile"
	FieldChapter   = "chapter"
)

// Author represents a person who contributed to an entry.
//
// Bibtex recognizes three structures for authors:
// 1. First von Last - no commas
// 2. First Last - no commas and no lowercase strings
// 3. von Last, First - single comma
// 4. von Last, Jr ,First - two commas
//
// Other parsing libraries:
// - https://metacpan.org/pod/distribution/Text-BibTeX/btparse/doc/bt_split_names.pod
// - https://nzhagen.github.io/bibulous/developer_guide.html#name-formatting
type Author struct {
	First  string // given name
	Prefix string // often called the 'von' part
	Last   string // family name
	Suffix string // often called the 'jr' part
}

func (a Author) IsOthers() bool {
	return a.First == "" && a.Prefix == "" && a.Last == "others" && a.Suffix == ""
}

type Entry struct {
	Type   EntryType
	Key    CiteKey
	Author []Author
	Editor []Author
	Title  string
	Tags   map[Field]string
}

func Read(r io.Reader) ([]Entry, error) {
	entries, err := ResolveFile(gotok.NewFileSet(), "", r)
	return entries, err
}

func IsValidCiteChar(ch byte) bool {
	return scanner.IsName(rune(ch))
}
