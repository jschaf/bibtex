package parser

import (
	"fmt"
	goscan "go/scanner"
	gotok "go/token"
	"strings"

	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/scanner"
	"github.com/jschaf/bibtex/token"
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
		m |= scanner.ScanComments
	}
	if mode&ParseStrings != 0 {
		m |= scanner.ScanStrings
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
		case p.tok.IsStringLiteral():
			lit := p.lit
			// Simplify trace expression.
			if lit != "" {
				lit = `"` + lit + `"`
			}
			p.printTrace(s, lit)
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

func (p *parser) expectOptional(tok token.Token) gotok.Pos {
	pos := p.pos
	if p.tok != tok {
		return pos
	}
	p.next() // make progress
	return pos
}

func (p *parser) expectOne(tok ...token.Token) (token.Token, gotok.Pos) {
	pos := p.pos
	for _, t := range tok {
		if p.tok == t {
			p.next()
			return t, pos
		}
	}

	sb := strings.Builder{}
	sb.WriteString("one of [")
	for i, t := range tok {
		sb.WriteString("'" + t.String() + "'")
		if i < len(tok)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")
	p.errorExpected(pos, sb.String())
	p.next() // make progress
	return token.Illegal, pos
}

func (p *parser) expectOptionalTagComma() {
	if p.tok == token.RBrace || p.tok == token.RParen {
		// TextComma is optional before a closing ')' or '}'
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
			// progress. Otherwise, consume at least one token to
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

func (p *parser) parseBasicLit() (l ast.Expr) {
	switch p.tok {
	case token.BraceString, token.String:
		l = &ast.UnparsedText{
			ValuePos: p.pos,
			Type:     p.tok,
			Value:    p.lit,
		}
		p.next()
	case token.Number:
		l = &ast.Number{
			ValuePos: p.pos,
			Value:    p.lit,
		}
		p.next()

	case token.Ident:
		l = p.parseIdent()

	default:
		p.errorExpected(p.pos, "literal: number or string")
	}
	return
}

// parseMacroURL parses a TeX \url or \href macro. This is separate because
// urls use common LaTeX characters like ~ for non-breaking spaces.
func (p *parser) parseMacroURL(name string) ast.Expr {
	urlCmd := &ast.TextMacro{Cmd: p.pos, Name: name}
	p.next()
	p.expect(token.StringLBrace)
	sb := strings.Builder{}
	sb.Grow(32)
	for p.tok != token.StringRBrace && p.tok != token.StringSpace {
		sb.WriteString(p.lit)
		p.next()
	}
	urlCmd.Values = []ast.Expr{
		&ast.Text{ValuePos: p.pos, Value: sb.String()},
	}
	if p.tok == token.StringSpace {
		p.next()
	}
	urlCmd.RBrace = p.pos

	pos := p.pos
	if p.tok != token.StringRBrace {
		p.errorExpected(pos, "'"+token.StringRBrace.String()+"'")
	}
	return urlCmd
}

func (p *parser) parseText(depth int) (txt ast.Expr) {
	switch p.tok {
	case token.StringMath:
		txt = &ast.TextMath{ValuePos: p.pos, Value: p.lit}
	case token.StringHyphen:
		txt = &ast.TextHyphen{ValuePos: p.pos}
	case token.StringNBSP:
		txt = &ast.TextNBSP{ValuePos: p.pos}
	case token.StringContents:
		txt = &ast.Text{ValuePos: p.pos, Value: p.lit}
	case token.StringSpace:
		txt = &ast.TextSpace{ValuePos: p.pos, Value: p.lit}
	case token.StringComma:
		txt = &ast.TextComma{ValuePos: p.pos}
	case token.StringMacro:
		switch p.lit {
		case `url`, `href`:
			// Special case common macros.
			txt = p.parseMacroURL(p.lit)
		default:
			txt = &ast.TextMacro{Cmd: p.pos, Name: p.lit}
		}
	case token.StringBackslash:
		txt = &ast.TextEscaped{ValuePos: p.pos, Value: p.lit[1:]}
	case token.Illegal:
		txt = &ast.BadExpr{From: p.pos, To: p.pos}
	case token.StringLBrace: // recursive case
		opener := p.pos
		p.next()

		values := make([]ast.Expr, 0, 2)
		for p.tok != token.StringRBrace {
			text := p.parseText(depth + 1)
			if _, ok := text.(*ast.BadExpr); ok {
				p.next()
				return text
			}
			values = append(values, text)
		}
		p.next() // consume closing '}'
		return &ast.ParsedText{
			Depth:  depth,
			Opener: opener,
			Delim:  ast.BraceDelimiter,
			Values: values,
			Closer: p.pos,
		}
	default:
		p.error(p.pos, "unknown text type: "+p.tok.String())
	}

	p.next()
	return
}

func (p *parser) parseStringLiteral() ast.Expr {
	pos := p.pos
	switch tok := p.tok; tok {
	case token.DoubleQuote:
		p.next()
		values := make([]ast.Expr, 0, 2)
		for p.tok != token.DoubleQuote {
			if p.tok == token.EOF {
				p.errorExpected(p.pos, "double quote")
				return &ast.BadExpr{From: pos, To: p.pos}
			}
			values = append(values, p.parseText(1))
		}
		p.next() // consume closing '"'
		txt := &ast.ParsedText{
			Opener: pos,
			Depth:  0,
			Delim:  ast.QuoteDelimiter,
			Values: values,
			Closer: p.pos,
		}
		return txt

	case token.StringLBrace:
		return p.parseText(0)

	default:
		p.errorExpected(p.pos, "string literal")
		p.advance(stmtStart)
		return &ast.BadExpr{
			From: pos,
			To:   p.pos,
		}
	}
}

func (p *parser) parseURLStringLiteral() ast.Expr {
	sb := strings.Builder{}
	sb.Grow(32)
	pos := p.pos
	switch tok := p.tok; tok {
	case token.DoubleQuote:
		p.next()

		if p.tok == token.StringMacro {
			url := p.parseMacroURL(p.lit)
			p.expect(token.StringRBrace)
			p.expect(token.DoubleQuote)
			return &ast.ParsedText{
				Opener: pos,
				Depth:  0,
				Delim:  ast.QuoteDelimiter,
				Values: []ast.Expr{url},
				Closer: p.pos,
			}
		}

		for p.tok != token.DoubleQuote {
			if p.tok == token.EOF {
				p.errorExpected(p.pos, "double quote")
				return &ast.BadExpr{From: pos, To: p.pos}
			}

			sb.WriteString(p.lit)
			p.next()
		}
		txt := &ast.Text{
			ValuePos: pos,
			Value:    sb.String(),
		}
		p.expect(token.DoubleQuote)
		return txt

	case token.StringLBrace:
		return p.parseText(0)

	default:
		p.errorExpected(p.pos, "string literal")
		p.advance(stmtStart)
		return &ast.BadExpr{
			From: pos,
			To:   p.pos,
		}
	}
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

	case p.tok.IsStringLiteral():
		x = p.parseStringLiteral()

	default:
		p.errorExpected(p.pos, "literal: number or string")
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
	p.expect(token.Assign)
	val := p.parseExpr()
	p.expectOptionalTagComma()
	return &ast.TagStmt{
		Doc:     doc,
		NamePos: key.Pos(),
		Name:    strings.ToLower(key.Name),
		RawName: key.Name,
		Value:   val,
	}
}

func (p *parser) expectCloser(open token.Token) gotok.Pos {
	end := p.pos
	switch open {
	case token.LBrace:
		end = p.expect(token.RBrace)
	case token.LParen:
		end = p.expect(token.RParen)
	default:
		p.error(p.pos, "no closing delimiter for "+open.String())
	}
	return end
}

func (p *parser) parsePreambleDecl() *ast.PreambleDecl {
	if p.trace {
		defer un(trace(p, "PreambleDecl"))
	}
	doc := p.leadComment
	pos := p.expect(token.Preamble)
	opener, _ := p.expectOne(token.LBrace, token.LParen)
	text := p.parseExpr()
	closer := p.expectCloser(opener)
	return &ast.PreambleDecl{
		Doc:    doc,
		Entry:  pos,
		Text:   text,
		RBrace: closer,
	}
}

func (p *parser) parseAbbrevDecl() *ast.AbbrevDecl {
	if p.trace {
		defer un(trace(p, "AbbrevDecl"))
	}
	doc := p.leadComment
	pos := p.expect(token.Abbrev)
	opener, _ := p.expectOne(token.LBrace, token.LParen)
	tag := p.parseTagStmt()
	closer := p.expectCloser(opener)
	return &ast.AbbrevDecl{
		Doc:    doc,
		Entry:  pos,
		Tag:    tag,
		RBrace: closer,
	}
}

// fixUpFields alters val based on tag type. For example, a url tag doesn't
// follow the normal Bibtex parsing rules because it's usually wrapped in a
// \url{} macro.
func fixUpFields(tag string, val ast.Expr) ast.Expr {
	if tag == "url" {
		txt, ok := val.(*ast.ParsedText)
		if !ok || len(txt.Values) == 0 {
			return val
		}
		child1, ok := txt.Values[0].(*ast.Text)
		if !ok || !strings.HasPrefix(child1.Value, "http") {
			return val
		}
		pos := child1.ValuePos
		sb := strings.Builder{}
		sb.Grow(32)
		for _, child := range txt.Values {
			if cTxt, ok := child.(*ast.Text); !ok {
				return val
			} else {
				sb.WriteString(cTxt.Value)
			}
		}
		return &ast.ParsedText{
			Opener: txt.Opener,
			Depth:  txt.Depth,
			Delim:  txt.Delim,
			Values: []ast.Expr{&ast.Text{ValuePos: pos, Value: sb.String()}},
			Closer: txt.Closer,
		}
	}

	return val
}

func (p *parser) parseBibDecl() *ast.BibDecl {
	if p.trace {
		defer un(trace(p, "BibDecl"))
	}
	doc := p.leadComment
	entryType := p.lit[1:] // drop '@', e.g. "@book" -> "book"
	pos := p.expect(token.BibEntry)
	var bibKey *ast.Ident // use first key found as bibKey
	var extraKeys []*ast.Ident
	tags := make([]*ast.TagStmt, 0, 8)
	opener, _ := p.expectOne(token.LBrace, token.LParen)
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
			var val ast.Expr
			if key.Name == "url" && p.tok.IsStringLiteral() {
				val = p.parseURLStringLiteral()
			} else {
				val = p.parseExpr()
			}
			fixVal := fixUpFields(key.Name, val)
			tag := &ast.TagStmt{
				Doc:     doc,
				NamePos: key.Pos(),
				Name:    strings.ToLower(key.Name),
				RawName: key.Name,
				Value:   fixVal,
			}
			tags = append(tags, tag)
		default:
			// Keep going.
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
		default:
			// Keep going.
		}
	}
	closer := p.expectCloser(opener)
	p.expectOptional(token.Comma) // trailing commas allowed
	return &ast.BibDecl{
		Type:      entryType,
		Doc:       doc,
		Entry:     pos,
		Key:       bibKey,
		ExtraKeys: extraKeys,
		Tags:      tags,
		RBrace:    closer,
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
	for p.tok != token.EOF && p.tok != token.Illegal {
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
