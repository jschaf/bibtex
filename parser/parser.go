package parser

import (
	"fmt"
	goscan "go/scanner"
	gotok "go/token"
	"strings"

	"github.com/jschaf/b2/pkg/bibtex/ast"
	"github.com/jschaf/b2/pkg/bibtex/scanner"
	"github.com/jschaf/b2/pkg/bibtex/token"
)

// The parser structure holds the parser's internal state.
type parser struct {
	file    *gotok.File
	errors  goscan.ErrorList
	scanner scanner.Scanner

	// Tracing/debugging
	mode   Mode // parsing mode
	trace  bool // == (mode & Trace != 0)
	indent int  // indentation used for tracing output

	// Comments
	comments    []*ast.TexCommentGroup
	leadComment *ast.TexCommentGroup // last lead comment
	lineComment *ast.TexCommentGroup // last line comment

	// Next token
	pos gotok.Pos   // token position
	tok token.Token // one token look-ahead
	lit string      // token literal

	// Error recovery
	// (used to limit the number of calls to parser.advance
	// w/o making scanning progress - avoids potential endless
	// loops across multiple parser functions during error recovery)
	syncPos gotok.Pos // last synchronization position
	syncCnt int       // number of parser.advance calls without progress

	// Ordinary cite key scopes
	pkgScope   *ast.Scope   // pkgScope.Outer == nil
	topScope   *ast.Scope   // top-most scope; may be pkgScope
	unresolved []*ast.Ident // unresolved cite keys
}

func (p *parser) init(fset *gotok.FileSet, filename string, src []byte, mode Mode) {
	p.file = fset.AddFile(filename, -1, len(src))
	var m scanner.Mode
	if mode&ParseComments != 0 {
		m = scanner.ScanComments
	}
	eh := func(pos gotok.Position, msg string) { p.errors.Add(pos, msg) }
	p.scanner.Init(p.file, src, eh, m)

	p.mode = mode
	p.trace = mode&Trace != 0 // for convenience (p.trace is used frequently)

	p.next()
}

// ----------------------------------------------------------------------------
// Scoping support

func (p *parser) openScope() {
	p.topScope = ast.NewScope(p.topScope)
}

func (p *parser) closeScope() {
	p.topScope = p.topScope.Outer
}

// The unresolved object is a sentinel to mark identifiers that have been added
// to the list of unresolved identifiers. The sentinel is only used for
// verifying internal consistency.
var unresolved = new(ast.Object)

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) printTrace(a ...interface{}) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	const n = len(dots)
	pos := p.file.Position(p.pos)
	fmt.Printf("%5d:%3d: ", pos.Line, pos.Column)
	i := 2 * p.indent
	for i > n {
		fmt.Print(dots)
		i -= n
	}
	// i <= n
	fmt.Print(dots[0:i])
	fmt.Println(a...)
}

func trace(p *parser, msg string) *parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p *parser) {
	p.indent--
	p.printTrace(")")
}

// Advance to the next token.
func (p *parser) next0() {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is token.ILLEGAL), so don't print it.
	if p.trace && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.IsLiteral():
			p.printTrace(s, p.lit)
		case p.tok.IsOperator(), p.tok.IsCommand():
			p.printTrace("\"" + s + "\"")
		default:
			p.printTrace(s)
		}
	}

	p.pos, p.tok, p.lit = p.scanner.Scan()
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (comment *ast.TexComment, endLine int) {
	endLine = p.file.Line(p.pos)
	comment = &ast.TexComment{Start: p.pos, Text: p.lit}
	p.next0()

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
func (p *parser) consumeCommentGroup(n int) (comments *ast.TexCommentGroup, endLine int) {
	var list []*ast.TexComment
	endLine = p.file.Line(p.pos)
	for p.tok == token.TexComment && p.file.Line(p.pos) <= endLine+n {
		var comment *ast.TexComment
		comment, endLine = p.consumeComment()
		list = append(list, comment)
	}

	// add comment group to the comments list
	comments = &ast.TexCommentGroup{List: list}
	p.comments = append(p.comments, comments)

	return
}

// Advance to the next non-comment token. In the process, collect
// any comment groups encountered, and remember the last lead and
// line comments.
//
// A lead comment is a comment group that starts and ends in a
// line without any other tokens and that is followed by a non-comment
// token on the line immediately after the comment group.
//
// A line comment is a comment group that follows a non-comment
// token on the same line, and that has no tokens after it on the line
// where it ends.
//
// Lead and line comments may be considered documentation that is
// stored in the AST.
//
func (p *parser) next() {
	p.leadComment = nil
	p.lineComment = nil
	prev := p.pos
	p.next0()

	if p.tok == token.TexComment {
		var comment *ast.TexCommentGroup
		var endLine int

		if p.file.Line(p.pos) == p.file.Line(prev) {
			// The comment is on same line as the previous token; it
			// cannot be a lead comment but may be a line comment.
			comment, endLine = p.consumeCommentGroup(0)
			if p.file.Line(p.pos) != endLine || p.tok == token.EOF {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endLine = -1
		for p.tok == token.TexComment {
			comment, endLine = p.consumeCommentGroup(1)
		}

		if endLine+1 == p.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}
}

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) error(pos gotok.Pos, msg string) {
	epos := p.file.Position(pos)

	// If AllErrors is not set, discard errors reported on the same line
	// as the last recorded error and stop parsing if there are more than
	// 10 errors.
	if p.mode&AllErrors == 0 {
		n := len(p.errors)
		if n > 0 && p.errors[n-1].Pos.Line == epos.Line {
			return // discard - likely a spurious error
		}
		if n > 10 {
			panic(bailout{})
		}
	}

	p.errors.Add(epos, msg)
}

func (p *parser) errorExpected(pos gotok.Pos, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		switch {
		case p.tok.IsLiteral():
			// print 123 rather than 'Number', etc.
			msg += ", found " + p.lit
		default:
			msg += ", found '" + p.tok.String() + "'"
		}
	}
	p.error(pos, msg)
}

func (p *parser) expect(tok token.Token) gotok.Pos {
	pos := p.pos
	if p.tok != tok {
		p.errorExpected(pos, "'"+tok.String()+"'")
	}
	p.next() // make progress
	return pos
}

func (p *parser) expectComma() {
	if p.tok == token.RBrace || p.tok == token.RParen {
		// Comma is optional before a closing ')' or '}'
		return
	}
	switch p.tok {
	case token.Comma:
		p.next()
	default:
		p.errorExpected(p.pos, "','")
		p.advance(stmtStart)
	}
}

func assert(cond bool, msg string) {
	if !cond {
		panic("bibtex/parser internal error: " + msg)
	}
}

// advance consumes tokens until the current token p.tok
// is in the 'to' set, or token.EOF. For error recovery.
func (p *parser) advance(to map[token.Token]bool) {
	for ; p.tok != token.EOF; p.next() {
		if to[p.tok] {
			// Return only if parser made some progress since last
			// sync or if it has not reached 10 advance calls without
			// progress. Otherwise consume at least one token to
			// avoid an endless parser loop (it is possible that
			// both parseOperand and parseStmt call advance and
			// correctly do not advance, thus the need for the
			// invocation limit p.syncCnt).
			if p.pos == p.syncPos && p.syncCnt < 10 {
				p.syncCnt++
				return
			}
			if p.pos > p.syncPos {
				p.syncPos = p.pos
				p.syncCnt = 0
				return
			}
			// Reaching here indicates a parser bug, likely an
			// incorrect token list in this function, but it only
			// leads to skipping of possibly correct code if a
			// previous error is present, and thus is preferred
			// over a non-terminating parse.
		}
	}
}

var stmtStart = map[token.Token]bool{
	token.Abbrev:   true,
	token.Comment:  true,
	token.Preamble: true,
	token.BibEntry: true,
	token.Ident:    true,
}

var entryStart = map[token.Token]bool{
	token.Abbrev:   true,
	token.Comment:  true,
	token.Preamble: true,
	token.BibEntry: true,
}

// isValidTagName returns true if the ident is a valid tag name.
// Uses rules according to Biber which means a tag key is a Bibtex name with the
// extra condition that it must begin with a letter:
// https://metacpan.org/pod/release/AMBS/Text-BibTeX-0.66/btparse/doc/bt_language.pod
func isValidTagName(key *ast.Ident) bool {
	ch := key.Name[0]
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func (p *parser) parseBasicLit() (l *ast.BasicLit) {
	switch {
	case p.tok.IsLiteral():
		l = &ast.BasicLit{
			ValuePos: p.pos,
			Kind:     p.tok,
			Value:    p.lit,
		}
	default:
		p.errorExpected(p.pos, "literal: number or string")
	}
	p.next()
	return
}

func (p *parser) parseExpr() (x ast.Expr) {
	if p.trace {
		defer un(trace(p, "Expr"))
	}
	pos := p.pos
	switch {
	case p.tok.IsLiteral():
		x = p.parseBasicLit()
		if p.tok == token.Concat {
			p.next()
			opPos := p.pos
			y := p.parseExpr()
			x = &ast.ConcatExpr{
				X:     x,
				OpPos: opPos,
				Y:     y,
			}
		}

	default:
		p.errorExpected(p.pos, "literal: number or string (\"foo\" or {foo})")
		x = &ast.BadExpr{
			From: pos,
			To:   p.pos,
		}
		p.next() // make progress
	}
	return
}

func (p *parser) parseIdent() *ast.Ident {
	pos := p.pos
	name := "_"
	switch p.tok {
	// Bibtex cite keys may be all numbers, but tag keys may not. Allow either
	// here and check one level up.
	case token.Ident, token.Number:
		name = p.lit
		p.next()
	default:
		p.expect(token.Ident) // use expect() error handling
	}
	return &ast.Ident{NamePos: pos, Name: name}
}

func (p *parser) parseTagStmt() *ast.TagStmt {
	if p.trace {
		defer un(trace(p, "TagStmt"))
	}
	doc := p.leadComment
	key := p.parseIdent()
	if p.tok == token.Assign {
		p.next()
	} else {
		p.expect(token.Assign) // use expect() error handling
	}
	val := p.parseExpr()
	p.expectComma()
	return &ast.TagStmt{
		Doc:     doc,
		NamePos: key.Pos(),
		Name:    strings.ToLower(key.Name),
		RawName: key.Name,
		Value:   val,
	}
}

func (p *parser) parsePreambleDecl() *ast.PreambleDecl {
	if p.trace {
		defer un(trace(p, "PreambleDecl"))
	}
	doc := p.leadComment
	pos := p.expect(token.Preamble)
	p.expect(token.LBrace)
	text := p.parseExpr()
	rBrace := p.expect(token.RBrace)
	return &ast.PreambleDecl{
		Doc:    doc,
		Entry:  pos,
		Text:   text,
		RBrace: rBrace,
	}
}

func (p *parser) parseAbbrevDecl() *ast.AbbrevDecl {
	if p.trace {
		defer un(trace(p, "AbbrevDecl"))
	}
	doc := p.leadComment
	pos := p.expect(token.Abbrev)
	p.expect(token.LBrace)
	tag := p.parseTagStmt()
	rBrace := p.expect(token.RBrace)
	return &ast.AbbrevDecl{
		Doc:    doc,
		Entry:  pos,
		Tag:    tag,
		RBrace: rBrace,
	}
}

func (p *parser) parseBibDecl() *ast.BibDecl {
	if p.trace {
		defer un(trace(p, "BibDecl"))
	}
	doc := p.leadComment
	pos := p.expect(token.BibEntry)
	var bibKey *ast.Ident
	var extraKeys []*ast.Ident
	tags := make([]*ast.TagStmt, 0, 8)
	p.expect(token.LBrace)
	// A bibtex entry cite key may be all numbers but a tag key cannot.
	for p.tok == token.Ident || p.tok == token.Number {
		doc := p.leadComment
		key := p.parseIdent() // parses both ident and number

		switch p.tok {
		case token.Assign:
			// It's a tag.
			if !isValidTagName(key) {
				p.error(key.Pos(), "tag keys must not start with a number")
			}
			p.next()
			val := p.parseExpr()
			tag := &ast.TagStmt{
				Doc:     doc,
				NamePos: key.Pos(),
				Name:    strings.ToLower(key.Name),
				RawName: key.Name,
				Value:   val,
			}
			tags = append(tags, tag)
		}
		switch p.tok {
		case token.Comma:
			// It's a cite key.
			p.next()
			if bibKey == nil {
				bibKey = key
			} else {
				extraKeys = append(extraKeys, key)
			}
			continue
		}
	}
	rBrace := p.expect(token.RBrace)
	return &ast.BibDecl{
		Doc:       doc,
		Entry:     pos,
		Key:       bibKey,
		ExtraKeys: extraKeys,
		Tags:      tags,
		RBrace:    rBrace,
	}
}

func (p *parser) parseDecl() ast.Decl {
	if p.trace {
		defer un(trace(p, "Declaration"))
	}

	switch p.tok {
	case token.Preamble:
		return p.parsePreambleDecl()
	case token.Abbrev:
		return p.parseAbbrevDecl()
	case token.BibEntry:
		return p.parseBibDecl()
	default:
		pos := p.pos
		p.errorExpected(pos, "entry")
		p.advance(entryStart)
		return &ast.BadDecl{
			From: pos,
			To:   p.pos,
		}
	}
}

// ----------------------------------------------------------------------------
// Source files

func (p *parser) parseFile() *ast.File {
	if p.trace {
		defer un(trace(p, "File"))
	}

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a bibtex source file at all.
	if p.errors.Len() != 0 {
		return nil
	}

	// Opening comment
	doc := p.leadComment

	p.openScope()
	p.pkgScope = p.topScope
	var decls []ast.Decl
	for p.tok != token.EOF {
		decls = append(decls, p.parseDecl())
	}
	p.closeScope()
	assert(p.topScope == nil, "unbalanced scopes")

	// resolve global identifiers within the same file
	i := 0
	for _, ident := range p.unresolved {
		// i <= index for current ident
		assert(ident.Obj == unresolved, "object already resolved")
		ident.Obj = p.pkgScope.Lookup(ident.Name) // also removes unresolved sentinel
		if ident.Obj == nil {
			p.unresolved[i] = ident
			i++
		}
	}

	return &ast.File{
		Doc:        doc,
		Entries:    decls,
		Scope:      p.pkgScope,
		Unresolved: p.unresolved[0:i],
		Comments:   p.comments,
	}
}
