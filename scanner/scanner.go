// Package scanner implements a scanner for bibtex source text.
// It takes a []byte as a source which can then be tokenized
// through repeated calls to the Scan method.
package scanner

import (
	"fmt"
	gotok "go/token"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jschaf/bibtex/token"
)

const (
	eof = -1
)

// An ErrorHandler may be provided to Scanner.Init. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
type ErrorHandler func(pos gotok.Position, msg string)

// A Scanner holds the scanner's internal state while processing
// a given text. It can be allocated as part of another data
// structure but must be initialized via Init before use.
type Scanner struct {
	// immutable state
	file *gotok.File  // source file handle
	dir  string       // directory portion of file.Name()
	src  []byte       // source
	err  ErrorHandler // error reporting; or nil
	mode Mode         // scanning mode

	// scanning state
	ch         rune        // current character
	offset     int         // character offset
	rdOffset   int         // reading offset (position after current character)
	lineOffset int         // current line offset
	prev       token.Token // previous token
	endQuoteCh rune        // '"' or '}'
	braceDepth int         // the brace depth in a string; starts at 0

	// public state - ok to modify
	ErrorCount int // number of errors encountered
}

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= utf8.RuneSelf:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		s.ch = eof
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
	s.ErrorCount++
}

const bom = 0xFEFF // byte order mark, only permitted as the first character

// peek returns the byte following the most recently read character without
// advancing the scanner. If the scanner is at EOF, peek returns 0.
func (s *Scanner) peek() byte {
	if s.rdOffset < len(s.src) {
		return s.src[s.rdOffset]
	}
	return 0
}

// Mode is a set of flags (or 0).
// They control scanner behavior.
type Mode uint

const (
	ScanComments Mode = 1 << iota // return comments as Comment or TexComment tokens
	ScanStrings                   // tokenize the contents of bibtex strings
)

// Init prepares the scanner s to tokenize the text src by setting the
// scanner at the beginning of src. The scanner uses the file set file
// for position information, and it adds line information for each line.
// It is ok to re-use the same file when re-scanning the same file as
// line information which is already present is ignored. Init causes a
// panic if the file size does not match the src size.
//
// Calls to Scan will invoke the error handler err if they encounter a
// syntax error and err is not nil. Also, for each error encountered,
// the Scanner field ErrorCount is incremented by one. The mode parameter
// determines how comments are handled.
//
// Note that Init may call err if there is an error in the first character
// of the file.
func (s *Scanner) Init(file *gotok.File, src []byte, err ErrorHandler, mode Mode) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}
	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.err = err
	s.mode = mode

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0
	s.lineOffset = 0
	s.ErrorCount = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at the file beginning
	}
}

func (s *Scanner) errorf(offset int, format string, args ...interface{}) {
	s.error(offset, fmt.Sprintf(format, args...))
}

func (s *Scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\n' || s.ch == '\r' {
		s.next()
	}
}

func lower(ch rune) rune     { return ('a' - 'A') | ch } // returns lower-case ch if ch is an ASCII letter
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }

func isLetter(ch rune) bool {
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func IsAsciiLetter(ch rune) bool { return 'a' <= lower(ch) && lower(ch) <= 'z' }

// IsName returns true if the char is a valid bibtex cite char.
// Taken from the btparse docs:
// https://metacpan.org/pod/release/AMBS/Text-BibTeX-0.66/btparse/doc/bt_language.pod
// Includes letters, digits, underscores, hyphens and the following:
//
//	! $ & * + - . / : ; < > ? [ ] ^ _ ` |
func IsName(ch rune) bool {
	return ('a' <= ch && ch <= 'z') ||
		('A' <= ch && ch <= 'Z') ||
		('0' <= ch && ch <= '9') ||
		ch == '_' ||
		ch == '-' ||
		ch == '/' ||
		ch == '!' ||
		ch == '$' ||
		ch == '&' ||
		ch == '*' ||
		ch == '+' ||
		ch == '.' ||
		ch == ':' ||
		ch == ';' ||
		ch == '<' ||
		ch == '>' ||
		ch == '?' ||
		ch == '[' ||
		ch == ']' ||
		ch == '^' ||
		ch == '`' ||
		ch == '|'
}

func (s *Scanner) scanCommand() string {
	offs := s.offset - 1 // Already consumed @
	s.next()
	if !isLetter(s.ch) {
		s.error(s.offset, "expected letter after @ for a command")
	}

	for isLetter(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanIdent() string {
	offs := s.offset
	for IsName(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

// scanString parses a bibtex string delimited by double quotes.
func (s *Scanner) scanString() string {
	offs := s.offset

	for {
		ch := s.ch
		if ch < 0 {
			s.error(offs, "string literal in double quotes not terminated")
			break
		}
		s.next()
		if ch == '"' {
			break
		}
		if ch == '{' {
			s.scanBraceString()
		}
	}
	return string(s.src[offs : s.offset-1])
}

func (s *Scanner) scanNumber() string {
	offs := s.offset
	for isDecimal(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

// scanBraceString parses a bibtex string delimited by braces.
func (s *Scanner) scanBraceString() string {
	offs := s.offset

	for {
		ch := s.ch
		if ch < 0 {
			s.error(offs, "string literal in braces not terminated")
			break
		}
		s.next()
		if ch == '}' {
			break
		}
		if ch == '{' {
			s.next()
			s.scanBraceString()
		}
	}
	return string(s.src[offs : s.offset-1])
}

func (s *Scanner) scanTexComment() string {
	offs := s.offset - 1 // initial '%' already consumed
	for s.ch != '\n' && s.ch >= 0 {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanStringMath() (token.Token, string) {
	offs := s.offset
	for s.ch != '$' {
		if s.ch < 0 || s.ch == '\n' {
			s.error(offs, "math in string literal not terminated")
			return token.Illegal, string(s.src[offs-1 : s.offset])
		}
		if s.ch == '\\' {
			s.next() // consume the backslash and whatever comes next
		}
		s.next()
	}
	s.next() // consume closing '$'
	return token.StringMath, string(s.src[offs : s.offset-1])
}

// scanStringEscape scans a string beginning with a backslash.
//
// A string beginning with a backslash can be an:
// - escape sequence for bibtex chars like \{ and \}
// - beginning of a latex command like \url{www.example.com}
// - beginning of a character accent like \^o to represent ô.
//
// See https://tex.stackexchange.com/a/66671/59048.
func (s *Scanner) scanStringEscape() (token.Token, string) {
	offs := s.offset - 1 // initial backslash already consumed
	switch s.ch {
	case '\\', '$', '&', '%', '{', '}', '_':
		// a single non-alphabetical character
		s.next()
		return token.StringBackslash, string(s.src[offs:s.offset])
	case rune(token.AccentAcute),
		rune(token.AccentCedilla),
		rune(token.AccentCircumflex),
		rune(token.AccentDot),
		rune(token.AccentGrave),
		rune(token.AccentTilde),
		rune(token.AccentUmlaut):
		return s.scanSpecialCharStringAccent()
	case ',', ';', '[', ']', '(', ')':
		// any single non-alphabetical character can be macro.
		s.next()
		return token.StringMacro, string(s.src[offs:s.offset])
	}

	// It must be a macro made up of ascii letters.
	lo := s.offset
	for !s.isSpecialStringChar(s.ch) && s.ch != 0 {
		s.next()
	}
	name := string(s.src[lo:s.offset])
	if len(name) == 0 {
		s.error(offs, "expected macro name after backslash, got nothing")
		return token.Illegal, string(s.src[offs : s.offset-1])
	}
	// Check that it's only ascii letters.
	for _, c := range name {
		if !IsAsciiLetter(c) {
			s.errorf(offs, "expected command name to only contain ascii letters, got %q", name)
			return token.Illegal, string(s.src[offs : s.offset-1])
		}
	}
	return token.StringMacro, name
}

// scanSpecialCharStringAccent scans a string that begins with a backslash
// followed by a special char like \'o, \'{o} or \^{o}.
func (s *Scanner) scanSpecialCharStringAccent() (token.Token, string) {
	offs := s.offset - 1 // initial backslash already consumed
	s.next()             // consume accent marker, like '"' or '^'
	if s.ch == '{' {
		s.next() // consume left brace '{'
		if !IsAsciiLetter(s.ch) {
			s.errorf(offs, "expected braced ascii letter after accent sequence %q , got %q", string(s.src[offs:s.offset-1]), s.ch)
			return token.Illegal, ""
		}
		s.next() // consume the letter that's accented
		if s.ch != '}' {
			s.errorf(offs, "expected right brace after accent sequence %q , got %q", string(s.src[offs:s.offset-1]), s.ch)
			return token.Illegal, ""
		}
		s.next() // consume right brace
	} else if s.ch == ' ' { // handle implicit braces like '\c c'
		// Construct accent string e.g. '\c'
		substring := string(s.src[offs:s.offset])
		s.next() // consume space
		// append accented char, e.g. 'c'
		substring += string(s.src[s.offset])
		s.next() // consume the letter that's accented
		return token.StringAccent, substring
	} else {
		if !IsAsciiLetter(s.ch) {
			s.errorf(offs, "expected ascii letter after accent sequence %q , got %q", string(s.src[offs:s.offset-1]), s.ch)
			return token.Illegal, ""
		}
		s.next() // consume the letter that's accented
	}
	return token.StringAccent, string(s.src[offs:s.offset])
}

func (s *Scanner) isSpecialStringChar(ch rune) bool {
	if ch == '"' {
		// A double quote is only special at brace depth 0 when we started the
		// string with a double quote because it terminates the string.
		return s.braceDepth == 0 && s.endQuoteCh == '"'
	}
	return ch == '$' || ch == '{' || ch == '}' ||
		ch == eof || ch == ',' ||
		ch == '~' || // nbsp
		ch == '\\' || // escape chars or begin a macro
		ch == '\n' || ch == '\r' || ch == ' ' || ch == '\t' // white space
}

func (s *Scanner) scanStringContents() string {
	offs := s.offset
	for !s.isSpecialStringChar(s.ch) {
		if s.ch == '\\' {
			s.next() // consume the backslash and next char
		}
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanInString() (pos gotok.Pos, tok token.Token, lit string) {
	if s.endQuoteCh == 0 {
		panic("called scanInString but not in quote")
	}
	pos = s.file.Pos(s.offset)
	if !s.isSpecialStringChar(s.ch) {
		tok = token.StringContents
		lit = s.scanStringContents()
		return
	}

	// It's a special char.
	ch := s.ch
	s.next()
	switch ch {
	case '$':
		tok, lit = s.scanStringMath()
	case '"':
		if s.endQuoteCh == '"' && s.braceDepth == 0 {
			s.endQuoteCh = 0
			tok = token.DoubleQuote
		} else {
			tok = token.StringContents
			lit = `"`
		}
	case '\\':
		tok, lit = s.scanStringEscape()
	case '{':
		s.braceDepth += 1
		tok = token.StringLBrace
	case '}':
		tok = token.StringRBrace
		if s.endQuoteCh == '}' && s.braceDepth == 0 {
			s.endQuoteCh = 0
		} else {
			s.braceDepth -= 1
		}
	case ' ', '\r', '\n', '\t':
		tok = token.StringSpace
		s.skipWhitespace()
	case ',':
		tok = token.StringComma
		lit = ","
	case '~':
		tok = token.StringNBSP
		lit = "~"
	default:
		// next reports unexpected BOMs - don't repeat
		if ch != bom {
			s.errorf(s.file.Offset(pos), "illegal character %#U in string", ch)
		}
		tok = token.Illegal
		lit = string(ch)
	}
	return
}

// Scan scans the next token and returns the token position, the token, and its
// literal string if applicable. The source end is indicated by token.EOF.
//
// If the returned token is a literal (token.Ident, token.Number, token.String),
// command, or token.TexComment, the literal string has the corresponding value.
//
// If the returned token is token.Illegal, the literal string is the offending
// character.
//
// In all other cases, Scan returns an empty literal string.
//
// For more tolerant parsing, Scan will return a valid token if possible even
// if a syntax error was encountered. Thus, even if the resulting token sequence
// contains no illegal tokens, a client may not assume that no error has
// occurred. Instead, the client must check the scanner's ErrorCount or the
// number of error handler calls if there was one installed.
//
// Scan adds line information to the file with Init. Token positions are
// relative to the file.
func (s *Scanner) Scan() (pos gotok.Pos, tok token.Token, lit string) {
	if s.endQuoteCh == '}' || s.endQuoteCh == '"' {
		return s.scanInString()
	}

	s.skipWhitespace()
	pos = s.file.Pos(s.offset)

	switch ch := s.ch; {
	case isDecimal(ch):
		tok = token.Number
		lit = s.scanNumber()

	case IsName(ch):
		tok = token.Ident
		lit = s.scanIdent()

	default:
		s.next() // always make progress
		switch ch {
		case -1:
			tok = token.EOF
		case '"':
			if s.mode&ScanStrings != 0 {
				s.endQuoteCh = '"'
				tok = token.DoubleQuote
			} else {
				tok = token.String
				lit = s.scanString()
			}
		case ',':
			tok = token.Comma
		case '=':
			tok = token.Assign
		case '@':
			lit = s.scanCommand()
			switch {
			case strings.EqualFold("@comment", lit):
				tok = token.Comment
			case strings.EqualFold("@string", lit):
				tok = token.Abbrev
			case strings.EqualFold("@preamble", lit):
				tok = token.Preamble
			default:
				tok = token.BibEntry
			}
		case '{':
			// Use a heuristic to determine whether this brace is for declaration or
			// a brace string. If preceded by '=', it's a string for a tag. If
			// preceded by an LBrace, it's a value in a block like:
			//   @preamble { {foo} }
			if s.prev == token.Assign || s.prev == token.LBrace {
				if s.mode&ScanStrings != 0 {
					s.endQuoteCh = '}'
					tok = token.StringLBrace
				} else {
					tok = token.BraceString
					lit = s.scanBraceString()
				}
			} else {
				tok = token.LBrace
			}
		case '}':
			tok = token.RBrace
		case '%':
			tok = token.TexComment
			lit = s.scanTexComment()
		case '#':
			tok = token.Concat
		case '(':
			tok = token.LParen
		case ')':
			tok = token.RParen

		default:
			// next reports unexpected BOMs - don't repeat
			if ch != bom {
				s.errorf(s.file.Offset(pos), "illegal character %#U", ch)
			}
			tok = token.Illegal
			lit = string(ch)
		}
	}

	s.prev = tok
	return
}
