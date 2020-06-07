// Package ast declares the types used to represent syntax trees for bibtex
// files.
package ast

import (
	gotok "go/token"

	"github.com/jschaf/b2/pkg/bibtex/token"
)

type Node interface {
	Pos() gotok.Pos
	End() gotok.Pos
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

// A TexCommentGroup represents a sequence of comments with no other tokens and
// no empty lines between.
type TexCommentGroup struct {
	List []*TexComment // len(List) > 0
}

func (g *TexCommentGroup) Pos() gotok.Pos { return g.List[0].Pos() }
func (g *TexCommentGroup) End() gotok.Pos { return g.List[len(g.List)-1].End() }

// ----------------------------------------------------------------------------
// Expressions

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
		NamePos   gotok.Pos // identifier position
		Name      string    // identifier name
		IsNumeric bool      // true if the name consists solely of digits
		Obj       *Object   // denoted object; or nil
	}

	// A BasicLit noe represents literals of basic type.
	BasicLit struct {
		ValuePos gotok.Pos   // literal position
		Kind     token.Token // token.Number, token.String, token.BraceString
		Value    string      // literal string, e.g. 42, "foo", {bar}
	}
)

func (x *BadExpr) Pos() gotok.Pos { return x.From }
func (x *BadExpr) End() gotok.Pos { return x.To }
func (*BadExpr) exprNode()        {}

func (x *Ident) Pos() gotok.Pos { return x.NamePos }
func (x *Ident) End() gotok.Pos { return gotok.Pos(int(x.NamePos) + len(x.Name)) }
func (*Ident) exprNode()        {}

func (x *BasicLit) Pos() gotok.Pos { return x.ValuePos }
func (x *BasicLit) End() gotok.Pos { return gotok.Pos(int(x.ValuePos) + len(x.Value)) }
func (*BasicLit) exprNode()        {}

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
func (*BadStmt) stmtNode()        {}

func (x *TagStmt) Pos() gotok.Pos { return x.NamePos }
func (x *TagStmt) End() gotok.Pos { return x.Value.Pos() }
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
		Doc    *TexCommentGroup // associated documentation; or nil
		Entry  gotok.Pos        // position of the start token, e.g. "@article"
		Keys   []*Ident
		Tags   []*TagStmt // all tags in the declaration
		RBrace gotok.Pos  // position of the closing right brace token: "}".
	}

	// An PreambleDecl node represents a bibtex preamble, like:
	//   @PREAMBLE { "foo" }
	PreambleDecl struct {
		Doc    *TexCommentGroup // associated documentation; or nil
		Entry  gotok.Pos        // position of the "@PREAMBLE" token
		Text   *BasicLit        // The content of the preamble node
		RBrace gotok.Pos        // position of the closing right brace token: "}"
	}
)

func (e *BadDecl) Pos() gotok.Pos { return e.From }
func (e *BadDecl) End() gotok.Pos { return e.To }
func (*BadDecl) declNode()        {}

func (e *AbbrevDecl) Pos() gotok.Pos { return e.Entry }
func (e *AbbrevDecl) End() gotok.Pos { return e.RBrace }
func (*AbbrevDecl) declNode()        {}

func (e *BibDecl) Pos() gotok.Pos { return e.Entry }
func (e *BibDecl) End() gotok.Pos { return e.RBrace }
func (*BibDecl) declNode()        {}

func (e *PreambleDecl) Pos() gotok.Pos { return e.Entry }
func (e *PreambleDecl) End() gotok.Pos { return e.RBrace }
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

// A Package node represents a set of source files collectively representing
// a single, unified bibliography.
type Package struct {
	Name    string             // package name
	Scope   *Scope             // package scope across all files
	Imports map[string]*Object // map of package id -> package object
	Files   map[string]*File   // Go source files by filename
}

func (p *Package) Pos() gotok.Pos { return gotok.NoPos }
func (p *Package) End() gotok.Pos { return gotok.NoPos }
