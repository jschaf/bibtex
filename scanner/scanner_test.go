package scanner

import (
	"github.com/google/go-cmp/cmp"
	gotok "go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jschaf/bibtex/token"
)

type tokClass int

func (t tokClass) String() string {
	switch t {
	case special:
		return "special"
	case literal:
		return "literal"
	case command:
		return "command"
	case operator:
		return "operator"
	default:
		return "tokClass(" + strconv.Itoa(int(t)) + ")"
	}
}

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
	{token.Comment, "@Comment", command},
	{token.Abbrev, "@String", command},
	{token.Abbrev, "@STRING", command},
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
	{token.TexComment, "% foo\n", special},
	{token.LBrace, `{`, operator},
	{token.BraceString, `{{f}oo}`, literal}, // LBrace must precede brace string
	{token.Ident, "qux", literal},
	{token.Ident, "qux_2", literal},
	{token.Ident, "q!$&*+-/<.:;<>?[]^_`|", literal},
	// Operators and delimiters
	{token.Concat, "#", operator},
	{token.LParen, "(", operator},
	{token.RParen, ")", operator},
	{token.LBrace, `{`, operator},
	{token.RBrace, `}`, operator},
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

// Verify that calling Scan() provides the correct results.
func TestScanner_Scan(t *testing.T) {
	whitespaceLineCount := newlineCount(whitespace)

	// error handler
	eh := func(_ gotok.Position, msg string) {
		t.Errorf("error handler called (msg = %s)", msg)
	}

	// verify scan
	var s Scanner
	fset := gotok.NewFileSet()
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

			p, tok, lit := s.Scan()

			// check position
			if tok == token.EOF {
				// correction for EOF
				epos.Line = newlineCount(string(source))
				epos.Column = 2
			}
			pos := fset.Position(p)
			checkPosFilename(t, pos, epos, lit)
			checkPosOffset(t, pos, epos, lit)
			checkPosLine(t, pos, epos, lit)
			checkPosColumn(t, pos, epos, lit)
			checkPosToken(t, tok, e.tok, lit)
			checkPosTokenClass(t, tok, e.class, lit)

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

// stringTok represents a token that's used for testing.
type stringTok struct {
	t   token.Token
	lit string
	raw string
}

func (st stringTok) newlineCount() int {
	return newlineCount(st.raw)
}

func (st stringTok) size() int {
	return len(st.raw)
}

type errorCollector struct {
	cnt  int              // number of errors encountered
	msgs []string         // all error messages encountered
	pos  []gotok.Position // error positions encountered
}

func (ec *errorCollector) asHandler() ErrorHandler {
	return func(pos gotok.Position, msg string) {
		ec.cnt++
		ec.msgs = append(ec.msgs, msg)
		ec.pos = append(ec.pos, pos)
	}
}

func errs(es ...string) []string {
	return es
}

func tok(s string) stringTok {
	switch {
	case strings.TrimSpace(s) == "":
		return stringTok{t: token.StringSpace, lit: "", raw: s}
	case s == `"`:
		return stringTok{t: token.DoubleQuote, lit: ``, raw: `"`}
	case s == ",":
		return stringTok{t: token.StringComma, lit: "", raw: ","}
	case s == "~":
		return stringTok{t: token.StringNBSP, lit: "", raw: "~"}
	case s == "LBrace":
		return stringTok{t: token.LBrace, lit: "", raw: "{"}
	case s == "=":
		return stringTok{t: token.Assign, lit: "", raw: "="}
	case s == "{":
		return stringTok{t: token.StringLBrace, lit: "", raw: "{"}
	case s == "}":
		return stringTok{t: token.StringRBrace, lit: "", raw: "}"}
	case strings.HasPrefix(s, "$"):
		if !strings.HasSuffix(s, "$") || len(s) < 2 {
			panic("tok begins with $ but doesn't end with $")
		}
		return stringTok{t: token.StringMath, lit: s[1 : len(s)-1], raw: s}
	default:
		return stringTok{t: token.StringContents, lit: s, raw: s}
	}
}

// toks returns a slice of stringTok by converting each string t into a
// a stringTok via the tok function.
func toks(t ...string) []stringTok {
	ts := make([]stringTok, len(t))
	for i := 0; i < len(t); i++ {
		ts[i] = tok(t[i])
	}
	return ts
}

func TestScanner_Scan_scanInString(t *testing.T) {
	type testCase struct {
		lit  string
		toks []stringTok
		errs []string
	}
	tests := []testCase{
		// Simple
		{"", toks(), nil},
		{`""`, toks(), nil},
		{`{}`, toks(), nil},
		{`"\$"`, toks(`"`, `\$`, `"`), nil},
		// Extra leading '=' here and below because we differentiate a string brace
		// from a declaration brace using a heuristic. If preceded by '=' or '{'
		// assume a string brace. Use it for double quotes for alignment between
		// similar test cases.
		{`={\$}`, toks("=", `{`, `\$`, `}`), nil},
		{`{{\$}`, toks("LBrace", `{`, `\$`, `}`), nil},
		// Escaped dollar signs
		{`="$\$$"`, toks("=", `"`, `$\$$`, `"`), nil},
		{`={$\$$}`, toks("=", `{`, `$\$$`, `}`), nil},
		{`="\$\$"`, toks("=", `"`, `\$\$`, `"`), nil},
		{`={\$\$}`, toks("=", `{`, `\$\$`, `}`), nil},
		// Escaped backslashes and special chars
		{`="\\a"`, toks("=", `"`, `\\a`, `"`), nil},
		{`={\\a}`, toks("=", `{`, `\\a`, `}`), nil},
		{`="\{"`, toks("=", `"`, `\{`, `"`), nil},
		{`={\{}`, toks("=", `{`, `\{`, `}`), nil},
		{`="\}"`, toks("=", `"`, `\}`, `"`), nil},
		{`={\}}`, toks("=", `{`, `\}`, `}`), nil},
		{`="\%"`, toks("=", `"`, `\%`, `"`), nil},
		{`={\%}`, toks("=", `{`, `\%`, `}`), nil},
		// Whitespace
		{"\"a\nb\"", toks(`"`, "a", "\n", "b", `"`), nil},
		{"\"a \n \r \t b\"", toks(`"`, "a", " \n \r \t ", "b", `"`), nil},
		{"={a \n \r \t b}", toks("=", `{`, "a", " \n \r \t ", "b", `}`), nil},
		{`="a,b"`, toks("=", `"`, "a", ",", "b", `"`), nil},
		{`={a~b}`, toks("=", `{`, "a", "~", "b", `}`), nil},
		{`="a{z"}b."`, toks("=", `"`, "a", "{", `z"`, "}", "b.", `"`), nil},
		{`={a{z"}b"}`, toks("=", `{`, "a", "{", `z"`, "}", `b"`, `}`), nil},
		{`="{Fo}o"`, toks("=", `"`, "{", "Fo", "}", "o", `"`), nil},
		{`={{Fo}o}`, toks(`=`, "{", "{", "Fo", "}", "o", `}`), nil},
		// Invalid
		{`"{ {$x}"`, []stringTok{tok(`"`), tok("{"), tok(" "), tok("{"),
			{t: token.Illegal, lit: `$x}"`, raw: `$x}"`},
		}, errs("math in string literal not terminated")},
	}

	for _, tt := range tests {
		t.Run(tt.lit, func(t *testing.T) {
			// error handler
			ec := &errorCollector{}

			// verify scanner
			fset := gotok.NewFileSet()
			var s Scanner
			s.Init(fset.AddFile("", fset.Base(), len(tt.lit)), []byte(tt.lit), ec.asHandler(), ScanStrings)

			// set up expected position
			epos := gotok.Position{
				Filename: "",
				Offset:   0,
				Line:     1,
				Column:   1,
			}

			for i := 0; i < len(tt.toks); i++ {
				p, tok, lit := s.Scan()
				eTok := tt.toks[i]
				t.Logf("index %2d, gotTok: %-14s wantTok: %-14s  gotLit: %-5q litAtPt: %q",
					i, tok, eTok.t, lit, eTok.raw)
				pos := fset.Position(p)
				checkPosFilename(t, pos, epos, lit)
				checkPosOffset(t, pos, epos, lit)
				// skip column check because no easy way to figure out expected column
				checkPosLine(t, pos, epos, lit)
				checkPosToken(t, tok, eTok.t, lit)

				if lit != eTok.lit {
					t.Errorf("bad literal for %q: got %q, expected %q",
						eTok.raw, lit, eTok.lit)
				}

				epos.Offset += eTok.size()
				epos.Line += eTok.newlineCount()
			}

			if diff := cmp.Diff(tt.errs, ec.msgs); diff != "" {
				t.Errorf("Errors mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestScanner_Scan_Errors(t *testing.T) {
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

	fset := gotok.NewFileSet()
	for _, e := range tests {
		t.Run(e.src, func(t *testing.T) {
			var s Scanner
			h := &errorCollector{}
			s.Init(fset.AddFile("", fset.Base(), len(e.src)), []byte(e.src), h.asHandler(), ScanComments)
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
			if e.err != "" {
				if diff := cmp.Diff(errs(e.err), h.msgs); diff != "" {
					t.Errorf("Errors mismatch (-want +got):\n%s", diff)
				}
			}
		})

	}
}

func checkPosTokenClass(t *testing.T, tok token.Token, e tokClass, lit string) {
	t.Helper()
	if tokenClass(tok) != e {
		t.Errorf("bad class for %q: got %s, expected %s", lit, tokenClass(tok), e)
	}
}

func checkPosToken(t *testing.T, tok, expected token.Token, lit string) {
	t.Helper()
	if tok != expected {
		t.Errorf("bad token for %q: got %s, expected %s", lit, tok, expected)
	}
}

func checkPosColumn(t *testing.T, pos gotok.Position, epos gotok.Position, lit string) {
	t.Helper()
	if pos.Column != epos.Column {
		t.Errorf("bad column for %q: got %d, expected %d", lit, pos.Column, epos.Column)
	}
}

func checkPosLine(t *testing.T, pos, expected gotok.Position, lit string) {
	t.Helper()
	if pos.Line != expected.Line {
		t.Errorf("bad line for %q: got %d, expected %d", lit, pos.Line, expected.Line)
	}
}

func checkPosOffset(t *testing.T, pos, expected gotok.Position, lit string) {
	t.Helper()
	if pos.Offset != expected.Offset {
		t.Errorf("bad position for %q: got %d, expected %d", lit, pos.Offset, expected.Offset)
	}
}

func checkPosFilename(t *testing.T, pos, expected gotok.Position, lit string) {
	t.Helper()
	// Check cleaned filenames so that we don't have to worry about
	// different os.PathSeparator values.
	if pos.Filename != expected.Filename && filepath.Clean(pos.Filename) != filepath.Clean(expected.Filename) {
		t.Errorf("bad filename for %q: got %s, expected %s", lit, pos.Filename, expected.Filename)
	}
}
