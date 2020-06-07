package scanner

import (
	gotok "go/token"
	"path/filepath"
	"testing"

	"github.com/jschaf/b2/pkg/bibtex/token"
)

var fset = gotok.NewFileSet()

type tokClass = int

const (
	special = iota
	literal
	command
	operator
)

func tokenClass(tok token.Token) tokClass {
	switch {
	case tok.IsLiteral():
		return literal
	case tok.IsOperator():
		return operator
	case tok.IsCommand():
		return command
	}
	return special
}

type elt struct {
	tok   token.Token
	lit   string
	class tokClass
}

var tokens = [...]elt{
	// Commands
	{token.Comment, "@COMMENT", command},
	{token.Comment, "@comMent", command},
	{token.Abbrev, "@String", command},
	{token.Abbrev, "@sTRING", command},
	{token.Preamble, "@preamble", command},
	{token.Preamble, "@PREAMBLE", command},
	{token.BibEntry, "@article", command},
	{token.BibEntry, "@ARTICLE", command},
	// Literals
	{token.String, `"foo"`, literal},
	{token.String, `""`, literal},
	{token.String, "\"\n\"", literal},
	{token.String, `"{"}"`, literal},
	{token.String, `"aa{"}bb{"}"`, literal},
	{token.Assign, `=`, operator},
	{token.BraceString, `{foo}`, literal}, // Assign must precede brace string
	{token.Assign, `=`, operator},
	{token.BraceString, `{{f}oo}`, literal}, // Assign must precede brace string
	{token.LBrace, `{`, operator},
	{token.RBrace, `}`, operator},
	{token.TexComment, "% foo\n", special},
	{token.LBrace, `{`, operator},
	{token.BraceString, `{{f}oo}`, literal}, // LBrace must precede brace string
	{token.Ident, "qux", literal},
	{token.Ident, "qux_2", literal},
	{token.Ident, "q!$&*+-./:;<>?[]^_`|", literal},
	// Operators
	{token.Concat, "#", operator},
}

const whitespace = "  \t  \n\n\n" // to separate tokens

var source = func() []byte {
	var src []byte
	for _, t := range tokens {
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

// Verify that calling Scan() provides the correct results.
func TestScan(t *testing.T) {
	whitespaceLineCount := newlineCount(whitespace)

	// error handler
	eh := func(_ gotok.Position, msg string) {
		t.Errorf("error handler called (msg = %s)", msg)
	}

	// verify scan
	var s Scanner
	s.Init(fset.AddFile("", fset.Base(), len(source)), source, eh, ScanComments)

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
		e := elt{token.EOF, "", special}
		if index < len(tokens) {
			e = tokens[index]
			index++
		}
		isDone := false
		t.Run(e.tok.String()+"-"+e.lit, func(t *testing.T) {

			pos, tok, lit := s.Scan()

			// check position
			if tok == token.EOF {
				// correction for EOF
				epos.Line = newlineCount(string(source))
				epos.Column = 2
			}
			checkPos(t, lit, pos, epos)

			if tok != e.tok {
				t.Errorf("bad token for %q: got %s, expected %s", lit, tok, e.tok)
			}

			// check token class
			if tokenClass(tok) != e.class {
				t.Errorf("bad class for %q: got %d, expected %d", lit, tokenClass(tok), e.class)
			}

			// check literal
			elit := ""
			switch e.tok {
			case token.BraceString, token.String:
				elit = e.lit
				elit = e.lit[1 : len(elit)-1] // Remove delimiters
			case token.Comment, token.Abbrev, token.BibEntry, token.Preamble:
				elit = e.lit
			case token.TexComment:
				elit = e.lit
				elit = elit[0 : len(elit)-1] // %-style comment doesn't contain newline.
			case token.Ident:
				elit = e.lit
			default:
				if e.tok.IsLiteral() {
					// no CRs in raw string literals
					elit = e.lit
				}
			}
			if lit != elit {
				t.Errorf("bad literal for %q: got %q, expected %q", lit, lit, elit)
			}

			if tok == token.EOF {
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

	if s.ErrorCount != 0 {
		t.Errorf("found %d errors", s.ErrorCount)
	}
}

func TestScanErrors(t *testing.T) {
	type errorCollector struct {
		cnt int            // number of errors encountered
		msg string         // last error message encountered
		pos gotok.Position // last error position encountered
	}

	tests := []struct {
		src string
		tok token.Token
		pos int
		lit string
		err string
	}{
		{`'`, token.Illegal, 0, `'`, "illegal character U+0027 '''"},
		// Valid
		{`""`, token.String, 0, ``, ""},
		{`"abc"`, token.String, 0, `abc`, ""},
		{`,`, token.Comma, 0, ``, ""},
		{`456`, token.Number, 0, `456`, ""},
	}
	for _, e := range tests {
		t.Run(e.src, func(t *testing.T) {
			var s Scanner
			var h errorCollector
			eh := func(pos gotok.Position, msg string) {
				h.cnt++
				h.msg = msg
				h.pos = pos
			}
			s.Init(fset.AddFile("", fset.Base(), len(e.src)), []byte(e.src), eh, ScanComments)
			_, tok0, lit0 := s.Scan()
			if tok0 != e.tok {
				t.Errorf("got %s, expected %s", tok0, e.tok)
			}
			if tok0 != token.Illegal && lit0 != e.lit {
				t.Errorf("got literal %q, expected %q", lit0, e.lit)
			}
			cnt := 0
			if e.err != "" {
				cnt = 1
			}
			if h.cnt != cnt {
				t.Errorf("got cnt %d, expected %d", h.cnt, cnt)
			}
			if h.msg != e.err {
				t.Errorf("got msg %q, expected %q", h.msg, e.err)
			}
			if h.pos.Offset != e.pos {
				t.Errorf("got offset %d, expected %d", h.pos.Offset, e.pos)
			}
		})

	}
}
