// Package scanner implements a scanner for bibtex source text.
// It takes a []byte as source which can then be tokenized
// through repeated calls to the Scan method.
package scanner

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jschaf/b2/pkg/bibtex/token"
)

// An ErrorHandler may be provided to Scanner.Init. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
type ErrorHandler func(pos token.Position, msg string)

// A Scanner holds the scanner's internal state while processing
// a given text. It can be allocated as part of another data
// structure but must be initialized via Init before use.
type Scanner struct {
	// immutable state
	file *token.File  // source file handle
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
		s.ch = -1 // eof
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
	s.ErrorCount++
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

// peek returns the byte following the most recently read character without
// advancing the scanner. If the scanner is at EOF, peek returns 0.
func (s *Scanner) peek() byte {
	if s.rdOffset < len(s.src) {
		return s.src[s.rdOffset]
	}
	return 0
}

// A mode value is a set of flags (or 0).
// They control scanner behavior.
type Mode uint

const (
	ScanComments Mode = 1 << iota // return comments as Comment or TexComment tokens
)

// Init prepares the scanner s to tokenize the text src by setting the
// scanner at the beginning of src. The scanner uses the file set file
// for position information and it adds line information for each line.
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
func (s *Scanner) Init(file *token.File, src []byte, err ErrorHandler, mode Mode) {
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
		s.next() // ignore BOM at file beginning
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

func lower(ch rune) rune     { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }

func isLetter(ch rune) bool {
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return isDecimal(ch) || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
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

// scanString parses an bibtex string delimited by double quotes.
func (s *Scanner) scanString() string {
	offs := s.offset - 1 // opening '"' already consumed

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
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanNumber() string {
	offs := s.offset - 1 // already scanned first digit
	for isDecimal(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

// scanBraceString parses an bibtex string delimited by braces.
func (s *Scanner) scanBraceString() string {
	offs := s.offset - 1 // opening '{' already consumed

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
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanTexComment() string {
	offs := s.offset - 1 // initial '%' already consumed
	for s.ch != '\n' && s.ch >= 0 {
		s.next()
	}
	return string(s.src[offs:s.offset])
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
// number of calls of the error handler, if there was one installed.
//
// Scan adds line information to the file with Init. Token positions are
// relative to the file.
func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	s.skipWhitespace()

	pos = s.file.Pos(s.offset)

	switch ch := s.ch; {
	default:
		s.next() // always make progress
		switch ch {
		case -1:
			tok = token.EOF
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			tok = token.Number
			lit = s.scanNumber()
		case '"':
			tok = token.String
			lit = s.scanString()
		case ',':
			tok = token.Comma
		case '=':
			tok = token.Assign
		case '@':
			lit = s.scanCommand()
			switch cmd := strings.ToLower(lit); cmd {
			case "@comment":
				tok = token.Comment
			case "@string":
				tok = token.Abbrev
			default:
				tok = token.Entry
			}
		case '{':
			if s.prev == token.Assign {
				tok = token.BraceString
				lit = s.scanBraceString()
			} else {
				tok = token.LBrace
			}
		case '}':
			tok = token.RBrace
		case '%':
			tok = token.TexComment
			lit = s.scanTexComment()

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
