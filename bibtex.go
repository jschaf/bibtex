package bibtex

import (
	"fmt"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/parser"
	"github.com/jschaf/bibtex/render"
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
	EntryDOI          Field = "doi"
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

// Bibtex contains methods for parsing and rendering bibtex.
type Bibtex struct {
	textRenderOverrides map[ast.TextKind]render.TextRendererFunc
	exprRenderer        render.ExprRenderer
}

type Option func(*Bibtex)

func WithTextRenderer(kind ast.TextKind, r render.TextRendererFunc) Option {
	return func(b *Bibtex) {
		b.textRenderOverrides[kind] = r
	}
}

func New(opts ...Option) *Bibtex {
	b := &Bibtex{
		textRenderOverrides: make(map[ast.TextKind]render.TextRendererFunc),
	}
	for _, opt := range opts {
		opt(b)
	}
	b.exprRenderer = render.NewTextRenderer(b.textRenderOverrides)
	return b
}

func (b *Bibtex) Parse(r io.Reader) (*ast.File, error) {
	f, err := parser.ParseFile(gotok.NewFileSet(), "", r, parser.ParseStrings)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (b *Bibtex) ResolveFile(f *ast.File) ([]Entry, error) {
	entries := make([]Entry, 0, len(f.Entries))
	for _, rawE := range f.Entries {
		if _, ok := rawE.(*ast.BibDecl); !ok {
			continue
		}
		bibDecl := rawE.(*ast.BibDecl)
		normE := Entry{
			Key:  bibDecl.Key.Name,
			Type: bibDecl.Type,
			Tags: make(map[Field]ast.Expr),
		}
		for _, tag := range bibDecl.Tags {
			normE.Tags[tag.Name] = tag.Value
		}
		entries = append(entries, normE)
	}
	return entries, nil
}

// ASTEntry is a Bibtex entry, like an @article{} entry, that provides AST for
// each tag in the entry.
type ASTEntry struct {
	// The type of entry, i.e. the "article" in @article{foo}.
	Type EntryType
	// The cite key of the entry, i.e. the "foo" in @article{foo, title = "bar"}.
	Key CiteKey
	// Map of the lowercase tag name to the expression for the tag.
	Tags map[Field]ast.Expr
}

// Entry is a Bibtex entry, like an @article{} entry, that provides the rendered
// plain text of the entry.
type Entry struct {
	Type EntryType
	Key  CiteKey
	// The parsed authors. The unparsed authors are available in
	// Tags[FieldAuthor]. Use a top-level field so users don't need an explicit
	// call to ResolveAuthors.
	Author []Author
	// The parsed editors. The unparsed editors are available in
	// Tags[FieldEditor].
	Editor []Author
	// All tags in the entry with the corresponding expression value.
	Tags map[Field]ast.Expr
}

// Parse reads all bibtex entries with the AST for each tag in the entry
// from the reader.
func Parse(r io.Reader) ([]ASTEntry, error) {
	entries, err := ResolveFile(gotok.NewFileSet(), "", r)
	return entries, err
}

// Read reads all bibtex entries as plain text from the reader.
func Read(r io.Reader) ([]Entry, error) {
	astEntries, err := Parse(r)
	if err != nil {
		return nil, fmt.Errorf("read bibtex entries: %w", err)
	}

	entries := make([]Entry, len(astEntries))
	renderer := render.NewTextRenderer()
	for i, astEntry := range astEntries {
		entry, err := renderEntryText(astEntry, renderer)
		if err != nil {
			return nil, fmt.Errorf("render bibtex entry as text: %w", err)
		}
		entries[i] = entry
	}
	return entries, nil
}
