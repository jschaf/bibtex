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
//
//	@article{ foo }
type CiteKey = string

// EntryType is the type of Bibtex entry. An "@article" entry is represented as
// "article". String alias to allow for unknown entries.
type EntryType = string

//goland:noinspection GoUnusedConst
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

//goland:noinspection GoUnusedConst
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

// Biber contains methods for parsing, resolving, and rendering bibtex.
type Biber struct {
	usePresets bool
	parserMode parser.Mode
	resolvers  []Resolver
	// Renderers for each node. The renderer for ast.Node n is contained at:
	//     renderers[n.Kind()]
	renderers []render.NodeRenderer
}

// Option is a functional option to change how Bibtex is parsed, resolved, and
// rendered.
type Option func(*Biber)

// WithParserMode sets the parser options overwriting any previous parser
// options. parser.Mode is a bitflag so use bit-or for multiple flags like so:
//
//	WithParserMode(parser.ParserStrings|parser.Trace)
func WithParserMode(mode parser.Mode) Option {
	return func(b *Biber) {
		b.parserMode = mode
	}
}

// WithResolvers appends the resolvers to the list of resolvers. Resolvers
// run in the order given.
func WithResolvers(rs ...Resolver) Option {
	return func(b *Biber) {
		for _, r := range rs {
			b.resolvers = append(b.resolvers, r)
		}
	}
}

// WithRenderer sets the renderer for the node kind, replacing the previous
// renderer.
func WithRenderer(kind ast.NodeKind, r render.NodeRendererFunc) Option {
	return func(b *Biber) {
		b.renderers[kind] = r
	}
}

func New(opts ...Option) *Biber {
	b := &Biber{
		// TODO: add mode to constant propagate abbrevs and concat expressions
		parserMode: parser.ParseStrings,
		renderers:  render.Defaults(),
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *Biber) Parse(r io.Reader) (*ast.File, error) {
	f, err := parser.ParseFile(gotok.NewFileSet(), "", r, b.parserMode)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Resolve resolves all bibtex entries from an AST. The AST is a faithful
// representation of source code. By default, resolving the AST means replacing
// all abbreviation expressions with the value, inlining concatenation
// expressions, simplifying tag values by replacing TeX quote macros with
// Unicode graphemes, and stripping Tex macros.
//
// The exact resolve steps are configurable using bibtex.WithResolvers.
func (b *Biber) Resolve(node ast.Node) ([]Entry, error) {
	for i, resolver := range b.resolvers {
		if err := resolver.Resolve(node); err != nil {
			return nil, fmt.Errorf("run resolvers[%d]: %w", i, err)
		}
	}
	switch n := node.(type) {
	case *ast.Package:
		entries := make([]Entry, 0, len(n.Files)*4)
		for _, file := range n.Files {
			for _, decl := range file.Entries {
				if decl, ok := decl.(*ast.BibDecl); ok {
					entries = append(entries, b.resolveEntry(decl))
				}
			}
		}
		return entries, nil

	case *ast.File:
		entries := make([]Entry, 0, len(n.Entries))
		for _, decl := range n.Entries {
			if decl, ok := decl.(*ast.BibDecl); ok {
				entries = append(entries, b.resolveEntry(decl))
			}
		}
		return entries, nil

	case *ast.BibDecl:
		return []Entry{b.resolveEntry(n)}, nil

	default:
		return nil, fmt.Errorf("bibtex.Resolve - node %T cannot be resolved into entries", node)
	}
}

func (b *Biber) resolveEntry(decl *ast.BibDecl) Entry {
	entry := Entry{
		Key:  decl.Key.Name,
		Type: decl.Type,
		Tags: make(map[Field]ast.Expr, 4),
	}
	for _, tag := range decl.Tags {
		entry.Tags[tag.Name] = tag.Value
	}
	return entry
}

// Entry is a Bibtex entry, like an @article{} entry, that provides the rendered
// plain text of the entry.
type Entry struct {
	Type EntryType
	Key  CiteKey
	// All tags in the entry with the corresponding expression value.
	Tags map[Field]ast.Expr
}

func (b *Biber) Render(w io.Writer, root ast.Node) error {
	err := ast.Walk(root, func(n ast.Node, isEntering bool) (ast.WalkStatus, error) {
		rend := b.renderers[n.Kind()]
		if rend == nil {
			return ast.WalkStop, fmt.Errorf("biber render - no renderer for node kind %s", n.Kind())
		}
		status, err := rend.Render(w, n, isEntering)
		if err != nil {
			return ast.WalkStop, fmt.Errorf("biber render - render error for node kind %s: %w", n.Kind(), err)
		}
		return status, nil
	})
	if err != nil {
		return fmt.Errorf("biber render - walk error: %w", err)
	}
	return nil
}
