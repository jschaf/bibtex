// Package ast declares the types used to represent syntax trees for bibtex
// files.
package ast

import (
	gotok "go/token"

	"github.com/jschaf/bibtex/token"
)

type Node interface {
	Pos() gotok.Pos
	End() gotok.Pos
	Kind() NodeKind
}

type NodeKind int

const (
	KindTexComment NodeKind = iota
	KindTexCommentGroup
	KindBadExpr
	KindIdent
	KindNumber
	KindAuthors
	KindAuthor
	KindUnparsedText
	KindParsedText
	KindText
	KindTextComma
	KindTextEscaped
	KindTextHyphen
	KindTextMath
	KindTextNBSP
	KindTextSpace
	KindTextMacro
	KindConcatExpr
	KindBadStmt
	KindTagStmt
	KindBadDecl
	KindAbbrevDecl
	KindBibDecl
	KindPreambleDecl
	KindFile
	KindPackage
)

var kindNames = [...]string{
	KindTexComment:      "TexComment",
	KindTexCommentGroup: "TexCommentGroup",
	KindBadExpr:         "BadExpr",
	KindIdent:           "Ident",
	KindNumber:          "Number",
	KindAuthors:         "Authors",
	KindAuthor:          "Author",
	KindUnparsedText:    "UnparsedText",
	KindParsedText:      "ParsedText",
	KindText:            "Text",
	KindTextComma:       "TextComma",
	KindTextEscaped:     "TextEscaped",
	KindTextHyphen:      "TextHyphen",
	KindTextMath:        "TextMath",
	KindTextNBSP:        "TextNBSP",
	KindTextSpace:       "TextSpace",
	KindTextMacro:       "TextMacro",
	KindConcatExpr:      "ConcatExpr",
	KindBadStmt:         "BadStmt",
	KindTagStmt:         "TagStmt",
	KindBadDecl:         "BadDecl",
	KindAbbrevDecl:      "AbbrevDecl",
	KindBibDecl:         "BibDecl",
	KindPreambleDecl:    "PreambleDecl",
	KindFile:            "File",
	KindPackage:         "Package",
}

func (k NodeKind) String() string {
	return kindNames[k]
}

// All expression nodes implement the Expr interface.
type Expr interface {
	Node
	exprNode()
}

// All statement nodes implement the Stmt interface, like bibtex entry tags.
type Stmt interface {
	Node
	stmtNode()
}

// All declaration nodes implement the Decl interface, like the @article,
// @STRING, @COMMENT, and @PREAMBLE entries.
type Decl interface {
	Node
	declNode()
}

// ----------------------------------------------------------------------------
// Comments

// A TexComment node represents a single %-style comment.
type TexComment struct {
	Start gotok.Pos // position of the '%' starting the comment
	Text  string    // comment text excluding '\n'
}

func (c *TexComment) Pos() gotok.Pos { return c.Start }
func (c *TexComment) End() gotok.Pos { return gotok.Pos(int(c.Start) + len(c.Text)) }
func (c *TexComment) Kind() NodeKind { return KindTexComment }

// A TexCommentGroup represents a sequence of comments with no other tokens and
// no empty lines between.
type TexCommentGroup struct {
	List []*TexComment // len(List) > 0
}

func (g *TexCommentGroup) Pos() gotok.Pos { return g.List[0].Pos() }
func (g *TexCommentGroup) End() gotok.Pos { return g.List[len(g.List)-1].End() }
func (g *TexCommentGroup) Kind() NodeKind { return KindTexCommentGroup }

// ----------------------------------------------------------------------------
// Expressions

type TextDelimiter int

func (t TextDelimiter) String() string {
	switch t {
	case QuoteDelimiter:
		return "QuoteDelimiter"
	case BraceDelimiter:
		return "BraceDelimiter"
	default:
		return "UnknownDelimiter"
	}
}

const (
	QuoteDelimiter TextDelimiter = iota
	BraceDelimiter
)

// An expression is represented by a tree consisting of one or more of the
// following concrete expressions.
type (
	// A BadExpr node is a placeholder for expressions containing syntax errors
	// for which no correct expression nodes can be created.
	BadExpr struct {
		From, To gotok.Pos
	}

	// An Ident node represents an identifier like a bibtex citation key or tag
	// key.
	Ident struct {
		NamePos gotok.Pos // identifier position
		Name    string    // identifier name
		Obj     *Object   // denoted object; or nil
	}

	// A Number node represents an unquoted number, like:
	//   year = 2004
	Number struct {
		ValuePos gotok.Pos
		Value    string
	}

	// An Authors node represents a list of authors, typically in the author or
	// editor fields of a bibtex declaration.
	Authors []*Author

	// An Author node represents a single bibtex author.
	Author struct {
		From, To gotok.Pos
		First    Expr // given name
		Prefix   Expr // often called the 'von' part
		Last     Expr // family name
		Suffix   Expr // often called the 'jr' part
	}

	// An UnparsedText is a bibtex string as it appears in source. Only appears
	// when Mode.ParseStrings == 0 is passed to ParseFile.
	UnparsedText struct {
		ValuePos gotok.Pos   // literal position
		Type     token.Token // token.String or token.BraceString
		Value    string      // excluding delimiters
	}

	// A ParsedText node represents a parsed bibtex string.
	ParsedText struct {
		Opener gotok.Pos // opening delimiter
		Depth  int       // the brace depth
		Delim  TextDelimiter
		Values []Expr    // Text, ParsedText, or any of the Text* types
		Closer gotok.Pos // closing delimiter
	}

	// A Text node is a string of simple text.
	Text struct {
		ValuePos gotok.Pos // literal position
		Value    string
	}

	// A TextComma node is a string of exactly 1 comma. Useful because a comma has
	// semantic meaning for parsing authors as a separator for names.
	TextComma struct {
		ValuePos gotok.Pos // literal position
	}

	// A TextEscaped node is a string of exactly 1 escaped character. The only
	// escapable characters are:
	//     '\\', '$', '&', '%', '{', '}'
	// In all other cases, a backslash is interpreted as the start of a TeX macro.
	TextEscaped struct {
		ValuePos gotok.Pos // literal position
		Value    string    // the escaped char without the backslash
	}

	// A TextHyphen node is a string of exactly 1 hyphen (-). Hyphens are
	// important in some cases of parsing author names so keep it as a separate
	// node.
	TextHyphen struct {
		ValuePos gotok.Pos // literal position
	}

	// A TextMath node is the string delimited by dollar signs, representing Math
	// in TeX.
	TextMath struct {
		ValuePos gotok.Pos // literal position
		Value    string    // the text in-between the $...$, not including the $'s.
	}

	// A TextNBSP node is a single non-breaking space, represented in TeX as '~'.
	TextNBSP struct {
		ValuePos gotok.Pos // literal position
	}

	// A TextSpace node is any consecutive whitespace '\n', '\r', '\t', ' ' in a
	// bibtex string.
	TextSpace struct {
		ValuePos gotok.Pos // literal position
		Value    string
	}

	// A TextMacro node represents a piece of ParsedText that's a latex macro
	// invocation.
	TextMacro struct {
		Cmd    gotok.Pos // command position
		Name   string    // command name without backslash, i.e. 'url'
		Values []Expr    // parameters: Text, ParsedText, or TextMacro
		RBrace gotok.Pos // position of the closing }, if any
	}

	// A ConcatExpr node represents a bibtex concatenation like:
	//   "foo" # "bar"
	ConcatExpr struct {
		X     Expr
		OpPos gotok.Pos
		Y     Expr
	}
)

func (x *BadExpr) Pos() gotok.Pos { return x.From }
func (x *BadExpr) End() gotok.Pos { return x.To }
func (x *BadExpr) Kind() NodeKind { return KindBadExpr }
func (*BadExpr) exprNode()        {}

func (x *Ident) Pos() gotok.Pos { return x.NamePos }
func (x *Ident) End() gotok.Pos { return gotok.Pos(int(x.NamePos) + len(x.Name)) }
func (x *Ident) Kind() NodeKind { return KindIdent }
func (*Ident) exprNode()        {}

func (x *Number) Pos() gotok.Pos { return x.ValuePos }
func (x *Number) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len(x.Value)) }
func (x *Number) Kind() NodeKind { return KindNumber }
func (*Number) exprNode()        {}

func (x Authors) Pos() gotok.Pos {
	if len(x) == 0 {
		return gotok.NoPos
	} else {
		return x[0].From
	}
}
func (x Authors) End() gotok.Pos {
	if len(x) == 0 {
		return gotok.NoPos
	} else {
		return x[len(x)-1].To
	}
}
func (x Authors) Kind() NodeKind { return KindAuthors }
func (Authors) exprNode()        {}

func (x *Author) Pos() gotok.Pos { return x.From }
func (x *Author) End() gotok.Pos { return x.To }
func (x *Author) Kind() NodeKind { return KindAuthor }
func (x *Author) IsEmpty() bool {
	if s, ok := x.First.(*Text); !ok || s.Value != "" {
		return false
	}
	if s, ok := x.Prefix.(*Text); !ok || s.Value != "" {
		return false
	}
	if s, ok := x.Last.(*Text); !ok || s.Value != "" {
		return false
	}
	if s, ok := x.Suffix.(*Text); !ok || s.Value != "" {
		return false
	}
	return true
}

// IsOthers returns true if this author was created from the "and others"
// suffix in from authors field.
func (x *Author) IsOthers() bool {
	if s, ok := x.First.(*Text); !ok || s.Value != "" {
		return false
	}
	if s, ok := x.Prefix.(*Text); !ok || s.Value != "" {
		return false
	}
	if s, ok := x.Last.(*Text); !ok || s.Value != "others" {
		return false
	}
	if s, ok := x.Suffix.(*Text); !ok || s.Value != "" {
		return false
	}
	return true
}
func (x *Author) exprNode() {}

func (x *UnparsedText) Pos() gotok.Pos { return x.ValuePos }
func (x *UnparsedText) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len(x.Value)) }
func (x *UnparsedText) Kind() NodeKind { return KindUnparsedText }
func (*UnparsedText) exprNode()        {}

func (x *ParsedText) Pos() gotok.Pos { return x.Opener }
func (x *ParsedText) End() gotok.Pos {
	if len(x.Values) > 0 {
		return x.Values[len(x.Values)-1].Pos()
	}
	return x.Opener
}
func (x *ParsedText) Kind() NodeKind { return KindParsedText }
func (*ParsedText) exprNode()        {}

func (x *Text) Pos() gotok.Pos { return x.ValuePos }
func (x *Text) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len(x.Value)) }
func (x *Text) Kind() NodeKind { return KindText }
func (*Text) exprNode()        {}

func (x *TextComma) Pos() gotok.Pos { return x.ValuePos }
func (x *TextComma) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len(",")) }
func (x *TextComma) Kind() NodeKind { return KindTextComma }
func (*TextComma) exprNode()        {}

func (x *TextEscaped) Pos() gotok.Pos { return x.ValuePos }
func (x *TextEscaped) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len(x.Value) + len(`\`)) }
func (x *TextEscaped) Kind() NodeKind { return KindTextEscaped }
func (*TextEscaped) exprNode()        {}

func (x *TextHyphen) Pos() gotok.Pos { return x.ValuePos }
func (x *TextHyphen) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len("-")) }
func (x *TextHyphen) Kind() NodeKind { return KindTextHyphen }
func (*TextHyphen) exprNode()        {}

func (x *TextMath) Pos() gotok.Pos { return x.ValuePos }
func (x *TextMath) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + 2*len("$") + len(x.Value)) }
func (x *TextMath) Kind() NodeKind { return KindTextMath }
func (*TextMath) exprNode()        {}

func (x *TextNBSP) Pos() gotok.Pos { return x.ValuePos }
func (x *TextNBSP) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len("~")) }
func (x *TextNBSP) Kind() NodeKind { return KindTextNBSP }
func (*TextNBSP) exprNode()        {}

func (x *TextSpace) Pos() gotok.Pos { return x.ValuePos }
func (x *TextSpace) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len(x.Value)) }
func (x *TextSpace) Kind() NodeKind { return KindTextSpace }
func (*TextSpace) exprNode()        {}

func (x *TextMacro) Pos() gotok.Pos { return x.Cmd }
func (x *TextMacro) End() gotok.Pos {
	if x.RBrace != gotok.NoPos {
		return x.RBrace
	}
	if len(x.Values) == 0 {
		return x.Cmd
	}
	return x.Values[len(x.Values)-1].Pos()
}
func (x *TextMacro) Kind() NodeKind { return KindTextMacro }
func (*TextMacro) exprNode()        {}

func (x *ConcatExpr) Pos() gotok.Pos { return x.X.Pos() }
func (x *ConcatExpr) End() gotok.Pos { return x.Y.Pos() }
func (x *ConcatExpr) Kind() NodeKind { return KindConcatExpr }
func (*ConcatExpr) exprNode()        {}

// ----------------------------------------------------------------------------
// Statements

// An statement is represented by a tree consisting of one or more of the
// following concrete statement nodes.
type (
	// A BadStmt node is a placeholder for statements containing syntax errors
	// for which no correct statement nodes can be created.
	BadStmt struct {
		From, To gotok.Pos // position range of bad statement
	}

	// An TagStmt node represents a tag in an BibDecl or AbbrevDecl, i.e.
	// author = "foo".
	TagStmt struct {
		Doc     *TexCommentGroup // associated documentation; or nil
		NamePos gotok.Pos        // identifier position
		Name    string           // identifier name, normalized with lowercase
		RawName string           // identifier name as it appeared in source
		Value   Expr             // denoted expression
	}
)

func (x *BadStmt) Pos() gotok.Pos { return x.From }
func (x *BadStmt) End() gotok.Pos { return x.To }
func (x *BadStmt) Kind() NodeKind { return KindBadStmt }
func (*BadStmt) stmtNode()        {}

func (x *TagStmt) Pos() gotok.Pos { return x.NamePos }
func (x *TagStmt) End() gotok.Pos { return x.Value.Pos() }
func (x *TagStmt) Kind() NodeKind { return KindTagStmt }
func (*TagStmt) stmtNode()        {}

// ----------------------------------------------------------------------------
// Declarations

// An declaration is represented by one of the following declaration nodes.
type (
	// A BadDecl node is a placeholder for declarations containing syntax errors
	// for which no correct declaration nodes can be created.
	BadDecl struct {
		From, To gotok.Pos // position range of bad declaration
	}

	// An AbbrevDecl node represents a bibtex abbreviation, like:
	//   @STRING { foo = "bar" }
	AbbrevDecl struct {
		Doc    *TexCommentGroup // associated documentation; or nil
		Entry  gotok.Pos        // position of the "@STRING" token
		Tag    *TagStmt
		RBrace gotok.Pos // position of the closing right brace token: "}".
	}

	// An BibDecl node represents a bibtex entry, like:
	//   @article { author = "bar" }
	BibDecl struct {
		Type      string           // type of entry, e.g. "article"
		Doc       *TexCommentGroup // associated documentation; or nil
		Entry     gotok.Pos        // position of the start token, e.g. "@article"
		Key       *Ident           // the first key in the declaration
		ExtraKeys []*Ident         // any other keys in the declaration, usually nil
		Tags      []*TagStmt       // all tags in the declaration
		RBrace    gotok.Pos        // position of the closing right brace token: "}".
	}

	// An PreambleDecl node represents a bibtex preamble, like:
	//   @PREAMBLE { "foo" }
	PreambleDecl struct {
		Doc    *TexCommentGroup // associated documentation; or nil
		Entry  gotok.Pos        // position of the "@PREAMBLE" token
		Text   Expr             // The content of the preamble node
		RBrace gotok.Pos        // position of the closing right brace token: "}"
	}
)

func (e *BadDecl) Pos() gotok.Pos { return e.From }
func (e *BadDecl) End() gotok.Pos { return e.To }
func (e *BadDecl) Kind() NodeKind { return KindBadDecl }
func (*BadDecl) declNode()        {}

func (e *AbbrevDecl) Pos() gotok.Pos { return e.Entry }
func (e *AbbrevDecl) End() gotok.Pos { return e.RBrace }
func (e *AbbrevDecl) Kind() NodeKind { return KindAbbrevDecl }
func (*AbbrevDecl) declNode()        {}

func (e *BibDecl) Pos() gotok.Pos { return e.Entry }
func (e *BibDecl) End() gotok.Pos { return e.RBrace }
func (e *BibDecl) Kind() NodeKind { return KindBibDecl }
func (*BibDecl) declNode()        {}

func (e *PreambleDecl) Pos() gotok.Pos { return e.Entry }
func (e *PreambleDecl) End() gotok.Pos { return e.RBrace }
func (e *PreambleDecl) Kind() NodeKind { return KindPreambleDecl }
func (*PreambleDecl) declNode()        {}

// ----------------------------------------------------------------------------
// Files and packages

// A File node represents a bibtex source file.
//
// The Comments list contains all comments in the source file in order of
// appearance, including the comments that are pointed to from other nodes
// via Doc and Comment fields.
//
// For correct printing of source code containing comments (using packages
// go/format and go/printer), special care must be taken to update comments
// when a File's syntax tree is modified: For printing, comments are interspersed
// between tokens based on their position. If syntax tree nodes are
// removed or moved, relevant comments in their vicinity must also be removed
// (from the File.Comments list) or moved accordingly (by updating their
// positions). A CommentMap may be used to facilitate some of these operations.
//
// Whether and how a comment is associated with a node depends on the
// interpretation of the syntax tree by the manipulating program: Except for Doc
// and Comment comments directly associated with nodes, the remaining comments
// are "free-floating".
type File struct {
	Name       string
	Doc        *TexCommentGroup   // associated documentation; or nil
	Entries    []Decl             // top-level entries; or nil
	Scope      *Scope             // package scope (this file only)
	Unresolved []*Ident           // unresolved identifiers in this file
	Comments   []*TexCommentGroup // list of all comments in the source file
}

func (f *File) Pos() gotok.Pos { return gotok.Pos(1) }
func (f *File) End() gotok.Pos {
	if n := len(f.Entries); n > 0 {
		return f.Entries[n-1].End()
	}
	return gotok.Pos(1)
}
func (f *File) Kind() NodeKind { return KindFile }

// A Package node represents a set of source files collectively representing
// a single, unified bibliography.
type Package struct {
	Scope   *Scope             // package scope across all files
	Objects map[string]*Object // map of package id -> package object
	Files   map[string]*File   // Bibtex source files by filename
}

func (p *Package) Pos() gotok.Pos { return gotok.NoPos }
func (p *Package) End() gotok.Pos { return gotok.NoPos }
func (p *Package) Kind() NodeKind { return KindPackage }
