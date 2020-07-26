package bibtex

import (
	"github.com/jschaf/bibtex/scanner"
	gotok "go/token"
	"io"
)

// CiteKey is the citation key for a Bibtex entry, like the "foo" in:
//   @article{ foo }
type CiteKey = string

// EntryType is the type of Bibtex entry. An "@article" entry is represented as
// "article". String alias to allow for unknown entries.
type EntryType = string

const (
	EntryArticle       EntryType = "article"
	EntryBook          EntryType = "book"
	EntryBooklet       EntryType = "booklet"
	EntryInBook        EntryType = "inbook"
	EntryInCollection  EntryType = "incollection"
	EntryInProceedings EntryType = "inproceedings"
	EntryManual        EntryType = "manual"
	EntryMastersThesis EntryType = "mastersthesis"
	EntryMisc          EntryType = "misc"
	EntryPhDThesis     EntryType = "phdthesis"
	EntryProceedings   EntryType = "proceedings"
	EntryTechReport    EntryType = "techreport"
	EntryUnpublished   EntryType = "unpublished"
)

// Field is a single field in a Bibtex Entry.
type Field = string

const (
	FieldAddress      Field = "address"
	FieldAnnote       Field = "annote"
	FieldAuthor       Field = "author"
	FieldBookTitle    Field = "booktitle"
	FieldChapter      Field = "chapter"
	FieldCrossref     Field = "crossref"
	FieldEdition      Field = "edition"
	FieldEditor       Field = "editor"
	FieldHowPublished Field = "howpublished"
	FieldInstitution  Field = "institution"
	FieldJournal      Field = "journal"
	FieldKey          Field = "key"
	FieldMonth        Field = "month"
	FieldNote         Field = "note"
	FieldNumber       Field = "number"
	FieldOrganization Field = "organization"
	FieldPages        Field = "pages"
	FieldPublisher    Field = "publisher"
	FieldSchool       Field = "school"
	FieldSeries       Field = "series"
	FieldTitle        Field = "title"
	FieldType         Field = "type"
	FieldVolume       Field = "volume"
	FieldYear         Field = "year"
)

// Author represents a person who contributed to an entry.
//
// Bibtex recognizes four structures for authors:
// 1. First von Last - no commas
// 2. First Last - no commas and no lowercase strings
// 3. von Last, First - single comma
// 4. von Last, Jr ,First - two commas
//
// This library treats "and others" as a special type of author recognized by
// the IsOthers() function.
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

// IsOthers returns true if this author was created from the "and others"
// suffix in from authors field.
func (a Author) IsOthers() bool {
	return a.First == "" && a.Prefix == "" && a.Last == "others" && a.Suffix == ""
}

// Entry is a Bibtex entry, like an @article{} entry.
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

// IsValidCiteChar returns true if ch is a valid character for a citation key.
func IsValidCiteChar(ch byte) bool {
	return scanner.IsName(rune(ch))
}
