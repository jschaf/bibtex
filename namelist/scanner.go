// Package namelist parses Bibtex style name lists.
package namelist

import (
	"github.com/jschaf/b2/pkg/bibtex/ast"
	goscan "go/scanner"
	gotok "go/token"
	"strings"
	"unicode/utf8"
)

type scanner struct {
	file       *gotok.File
	tok        ast.BasicLit
	src        []byte  // same as []byte(tok.Value), for convenience
	ch         rune    // current character
	rdOffset   int     // reading offset (position after current character)
	offset     int     // character offset
	lineOffset int     // current line offset
	prev       NameTok // previous token
	prev2      NameTok // previous-previous token

	nameSeps []string // separator strings, typically just "and"
	others   []string // additional author strings, typically just "others"

	errors goscan.ErrorList
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

func (s *scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.lineOffset = s.offset
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal char NUL in author string")
		case r >= utf8.RuneSelf:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding in author string")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark in author string")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.lineOffset = s.offset
		}
		s.ch = -1
	}

}

// Init prepares the scanner s to tokenize the text src by setting the scanner
// at the beginning of src. The scanner uses the file set file for position
// information and it adds line information for each line. It is ok to re-use
// the same file when re-scanning the same file as line information which is
// already present is ignored.
//
// Note that init may call err if there is an error in the first character
// of the file.
func (s *scanner) init(file *gotok.File, src []byte) {
	s.file = file
	s.src = src

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0
	s.lineOffset = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

func (s *scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\n' || s.ch == '\r' {
		s.next()
	}
}

func (s *scanner) scanString() string {
	offs := s.offset

	for s.ch != ' ' && s.ch != '\t' && s.ch != '\n' && s.ch != '\r' &&
		s.ch != '{' && s.ch != '}' && s.ch != ',' && s.ch > 0 {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func (s *scanner) error(offset int, msg string) {
	epos := s.file.Position(gotok.Pos(int(s.tok.Pos()) + offset))
	s.errors.Add(epos, msg)
}

func (s *scanner) isNameSep(o string) bool {
	for _, other := range s.nameSeps {
		if o == other {
			return true
		}
	}
	return false
}

func (s *scanner) isOthers(o string) bool {
	for _, other := range s.others {
		if o == other {
			return true
		}
	}
	return false
}

// scanBraceString parses an bibtex string delimited by braces.
func (s *scanner) scanBraceString() string {
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

func (s *scanner) scan() (pos gotok.Pos, tok NameTok, lit string) {
	s.skipWhitespace() // collapse adjacent whitespace
	pos = s.file.Pos(s.offset)

	switch ch := s.ch; ch {
	case -1:
		tok = EOF
	// case ' ', '\t', '\n', '\r':
	// 	tok = Whitespace
	// 	s.skipWhitespace() // collapse adjacent whitespace
	case '{':
		tok = BraceString
		lit = s.scanBraceString()
	case ',':
		tok = Comma
	default:
		tok = String
		lit = s.scanString()
		switch l := strings.ToLower(lit); {
		case s.prev == Whitespace && s.isNameSep(l):
			tok = NameSep
		case s.prev2 == NameSep && s.prev == Whitespace && s.isOthers(l):
			tok = Others
		}
	}

	s.prev2 = s.prev
	s.prev = tok
	return
}
