package bibtex

type CiteKey = string

type EntryType = string

const (
	Article EntryType = "article"
)

type Field = string

const (
	FieldAddress   = "address"
	FieldAuthor    = "author"
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
	First  string // aka given name
	Prefix string // often called the 'von' part
	Last   string // aka family name
	Suffix string // often called the 'jr' part
}

type Entry struct {
	Type    EntryType
	Key     CiteKey
	Authors []Author
	Editors []Author
	Title   string
	Tags    map[Field]string
}
