package namelist

import (
	gotok "go/token"
	"path/filepath"
	"testing"
)

var fset = gotok.NewFileSet()

type elt struct {
	tok NameTok
	lit string
}

var testTokens = [...]elt{
	// {Whitespace, " \n\r"},
	// {Whitespace, "\n \r"},
	{String, "foo"},
}

const whitespace = "  \t  \n\n\n" // to separate tokens

var source = func() []byte {
	var src []byte
	for _, t := range testTokens {
		src = append(src, t.lit...)
		src = append(src, whitespace...)
	}
	return src
}()

func newlineCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

func checkPos(t *testing.T, lit string, p gotok.Pos, expected gotok.Position) {
	pos := fset.Position(p)
	// Check cleaned filenames so that we don't have to worry about
	// different os.PathSeparator values.
	if pos.Filename != expected.Filename && filepath.Clean(pos.Filename) != filepath.Clean(expected.Filename) {
		t.Errorf("bad filename for %q: got %s, expected %s", lit, pos.Filename, expected.Filename)
	}
	if pos.Offset != expected.Offset {
		t.Errorf("bad position for %q: got %d, expected %d", lit, pos.Offset, expected.Offset)
	}
	if pos.Line != expected.Line {
		t.Errorf("bad line for %q: got %d, expected %d", lit, pos.Line, expected.Line)
	}
	if pos.Column != expected.Column {
		t.Errorf("bad column for %q: got %d, expected %d", lit, pos.Column, expected.Column)
	}
}

func TestScan(t *testing.T) {
	whitespaceLineCount := newlineCount(whitespace)
	var s scanner
	s.init(fset.AddFile("", fset.Base(), len(source)), source)

	// set up expected position
	epos := gotok.Position{
		Filename: "",
		Offset:   0,
		Line:     1,
		Column:   1,
	}

	index := 0

	for {
		// check token
		e := elt{EOF, ""}
		if index < len(testTokens) {
			e = testTokens[index]
			index++
		}
		isDone := false
		t.Run(e.tok.String()+"-"+e.lit, func(t *testing.T) {

			pos, tok, lit := s.scan()

			// check position
			if tok == EOF {
				// 	correction for EOF
				epos.Line = newlineCount(string(source))
				epos.Column = 2
			}
			checkPos(t, lit, pos, epos)

			if tok != e.tok {
				t.Errorf("bad token for %q: got %s, expected %s", e.lit, tok, e.tok)
			}

			// check literal
			elit := e.lit
			if lit != elit {
				t.Errorf("bad literal for %q: got %q, expected %q", e.lit, lit, elit)
			}

			if tok == EOF {
				isDone = true
			}

			// update position
			epos.Offset += len(e.lit) + len(whitespace)
			epos.Line += newlineCount(e.lit) + whitespaceLineCount

		})
		if isDone {
			break
		}
	}
}
