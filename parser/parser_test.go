package parser

import (
	"fmt"
	gotok "go/token"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/b2/pkg/bibtex/ast"
	"github.com/jschaf/b2/pkg/bibtex/token"
)

func cmpExpr() cmp.Option {
	return cmp.Transformer("expr_name", func(x ast.Expr) string {
		return exprString(x)
	})
}

func cmpIdentName() cmp.Option {
	return cmp.Transformer("ident_name", func(x *ast.Ident) string {
		return x.Name
	})
}

func exprString(x ast.Expr) string {
	switch v := x.(type) {
	case *ast.Ident:
		return "Ident(" + v.Name + ")"
	case *ast.Number:
		return "Number(" + v.Value + ")"
	case *ast.UnparsedText:
		if v.Kind == token.String {
			return "UnparsedText(\"" + v.Value + "\")"
		} else {
			return "UnparsedText({" + v.Value + "})"
		}
	case *ast.Text:
		return "Text[" + v.Kind.String() + "](" + v.Value + ")"

	case *ast.ParsedText:
		sb := strings.Builder{}
		sb.WriteString("ParsedText[" + strconv.Itoa(v.Depth) + "]")
		if v.Delim == ast.QuoteDelimiter {
			sb.WriteString(`"`)
		} else {
			sb.WriteString("{")
		}
		for i, val := range v.Values {
			sb.WriteString(exprString(val))
			if i < len(v.Values)-1 {
				sb.WriteString(", ")
			}
		}
		if v.Delim == ast.QuoteDelimiter {
			sb.WriteString(`")`)
		} else {
			sb.WriteString("})")
		}
		return sb.String()

	case *ast.ConcatExpr:
		return exprString(v.X) + " # " + exprString(v.Y)

	default:
		return fmt.Sprintf("UnknownExpr(%v)", v)
	}
}

func concat(x, y ast.Expr) ast.Expr {
	return &ast.ConcatExpr{X: x, Y: y}
}

func unparsedBraceText(s string) ast.Expr {
	return &ast.UnparsedText{
		Kind:  token.BraceString,
		Value: s,
	}
}

// bText return parsed text delimited by braces.
func bText(depth int, ss ...ast.Expr) ast.Expr {
	return &ast.ParsedText{
		Depth:  depth,
		Delim:  ast.BraceDelimiter,
		Values: ss,
	}
}

// qText return parsed text delimited by quotes.
func qText(depth int, ss ...ast.Expr) ast.Expr {
	return &ast.ParsedText{
		Depth:  depth,
		Delim:  ast.QuoteDelimiter,
		Values: ss,
	}
}

func text(kind ast.TextKind, s string) *ast.Text {
	return &ast.Text{
		Kind:  kind,
		Value: s,
	}
}

func txt(s string) *ast.Text {
	return &ast.Text{
		Kind:  ast.TextContent,
		Value: s,
	}
}

func wSpace() *ast.Text {
	return &ast.Text{Kind: ast.TextSpace}
}

func unparsedText(s string) ast.Expr {
	return &ast.UnparsedText{
		Kind:  token.String,
		Value: s,
	}
}

func ident(s string) ast.Expr {
	return &ast.Ident{
		Name: s,
		Obj:  nil,
	}
}

func cmpTagEntry() cmp.Option {
	return cmp.Transformer("tag_entry", func(t *ast.TagStmt) string {
		return t.RawName + " = " + exprString(t.Value)
	})
}

var validFiles = []string{
	"testdata/vldb.bib",
}

func TestParseFile_validFiles(t *testing.T) {
	for _, filename := range validFiles {
		_, err := ParseFile(gotok.NewFileSet(), filename, nil, DeclarationErrors)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", filename, err)
		}
	}
}

func BenchmarkParseFile_vldb(b *testing.B) {
	b.StopTimer()
	f, err := ioutil.ReadFile("testdata/vldb.bib")
	if err != nil {
		b.Fatalf("read file: %s", err.Error())
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseFile(gotok.NewFileSet(), "", f, 0)
		if err != nil {
			b.Fatalf(err.Error())
		}
	}
}

func TestParseFile_PreambleDecl(t *testing.T) {
	tests := []struct {
		src  string
		want ast.Expr
	}{
		{"@PREAMBLE { {foo} }", unparsedBraceText("foo")},
		{`@PREAMBLE { "foo" }`, unparsedText("foo")},
		{`@PREAMBLE ( "foo" )`, unparsedText("foo")},
		{`@preamble { "foo" }`, unparsedText("foo")},
		{`@preamble { "foo" # "bar" }`, concat(unparsedText("foo"), unparsedText("bar"))},
		{`@preamble { "foo" # "bar" # "qux" }`, concat(unparsedText("foo"), concat(unparsedText("bar"), unparsedText("qux")))},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err != nil {
				t.Fatal(err)
			}

			got := f.Entries[0].(*ast.PreambleDecl).Text

			if diff := cmp.Diff(tt.want, got, cmpExpr()); diff != "" {
				t.Errorf("PreambleDecl mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseFile_AbbrevDecl(t *testing.T) {
	tests := []struct {
		src     string
		tok     token.Token
		wantKey string
		wantVal ast.Expr
	}{
		{"@string { key = {foo} }", token.BraceString, "key", unparsedBraceText("foo")},
		{"@string { KeY = {foo} }", token.BraceString, "KeY", unparsedBraceText("foo")},
		{`@string { KeY = "foo" }`, token.String, "KeY", unparsedText("foo")},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err != nil {
				t.Fatal(err)
			}

			tag := f.Entries[0].(*ast.AbbrevDecl).Tag

			expected := &ast.TagStmt{
				Name:    strings.ToLower(tt.wantKey),
				RawName: tt.wantKey,
				Value:   tt.wantVal,
			}
			if diff := cmp.Diff(tag, expected, cmpTagEntry()); diff != "" {
				t.Errorf("BibDecl keys mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func bibKeys(ts ...string) func(decl *ast.BibDecl) {
	return func(b *ast.BibDecl) {
		if len(ts) > 0 {
			b.Key = &ast.Ident{Name: ts[0]}
			ts = ts[1:]
		}
		for _, k := range ts {
			b.ExtraKeys = append(b.ExtraKeys, &ast.Ident{Name: k})
		}
	}
}

func bibTags(key string, val ast.Expr, rest ...interface{}) func(decl *ast.BibDecl) {
	if len(rest)%2 != 0 {
		panic("bibTags must have even number of strings for key-val pairs")
	}
	for i := 0; i < len(rest); i += 2 {
		k := rest[i]
		v := rest[i+1]
		if _, ok := k.(string); !ok {
			panic("need string at index: " + strconv.Itoa(i))
		}
		if _, ok := v.(ast.Expr); !ok {
			panic(fmt.Sprintf("need ast.Expr at index: %d of bibTags, got: %v", i+1, v))
		}
	}
	return func(b *ast.BibDecl) {
		b.Tags = append(b.Tags, &ast.TagStmt{
			Name:    key,
			RawName: key,
			Value:   val,
		})
		for i := 0; i < len(rest); i += 2 {
			k, v := rest[i].(string), rest[i+1].(ast.Expr)
			tag := &ast.TagStmt{
				Name:    k,
				RawName: k,
				Value:   v,
			}
			b.Tags = append(b.Tags, tag)
		}
	}
}

func TestParseFile_BibDecl(t *testing.T) {
	tests := []struct {
		src    string
		keysFn func(*ast.BibDecl)
		tagsFn func(*ast.BibDecl)
	}{
		{"@article { cite_key, key = {foo} }", bibKeys("cite_key"), bibTags("key", unparsedBraceText("foo"))},
		{"@article {cite_key1, key = {foo} }", bibKeys("cite_key1"), bibTags("key", unparsedBraceText("foo"))},
		{"@article {111, key = {foo} }", bibKeys("111"), bibTags("key", unparsedBraceText("foo"))},
		{"@article {111, key = bar }", bibKeys("111"), bibTags("key", ident("bar"))},
		{"@article {111, key = bar, extra }", bibKeys("111", "extra"), bibTags("key", ident("bar"))},
		{`@article {111, key = bar, a, b, k2 = "v2" }`, bibKeys("111", "a", "b"), bibTags("key", ident("bar"), "k2", unparsedText("v2"))},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err != nil {
				t.Fatal(err)
			}

			gotBib := f.Entries[0].(*ast.BibDecl)
			wantBib := &ast.BibDecl{}
			tt.keysFn(wantBib)
			tt.tagsFn(wantBib)

			if diff := cmp.Diff(wantBib.Key, gotBib.Key, cmpIdentName()); diff != "" {
				t.Errorf("BibDecl keys mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantBib.Tags, gotBib.Tags, cmpTagEntry()); diff != "" {
				t.Errorf("BibDecl keys mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseFile_BibDecl_ModeParseStrings(t *testing.T) {
	tests := []struct {
		src    string
		keysFn func(*ast.BibDecl)
		tagsFn func(*ast.BibDecl)
	}{
		{"@article { cite_key, key = {foo} }", bibKeys("cite_key"), bibTags("key", bText(0, txt("foo")))},
		{"@article { cite_key, key = {{f}oo}}", bibKeys("cite_key"), bibTags("key", bText(0, bText(1, txt("f")), txt("oo")))},
		{`@article { cite_key, key = "foo" }`, bibKeys("cite_key"), bibTags("key", qText(0, txt("foo")))},
		{"@article { cite_key, key = {f\no\ro} }",
			bibKeys("cite_key"),
			bibTags("key", bText(0, txt("f"), wSpace(), txt("o"), wSpace(), txt("o")))},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, ParseStrings)
			if err != nil {
				t.Fatal(err)
			}

			gotBib := f.Entries[0].(*ast.BibDecl)
			wantBib := &ast.BibDecl{}
			tt.keysFn(wantBib)
			tt.tagsFn(wantBib)

			if diff := cmp.Diff(wantBib.Key, gotBib.Key, cmpIdentName()); diff != "" {
				t.Errorf("BibDecl keys mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantBib.Tags, gotBib.Tags, cmpTagEntry()); diff != "" {
				t.Errorf("BibDecl keys mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseFile_BibDecl_invalid(t *testing.T) {
	tests := []struct {
		src string
	}{
		{"@article {111, 111 = {foo} }"}, // tag keys must not be all numeric
		{"@article { foo = {foo} )"},     // mismatched delimiters
		{"@article ( foo = {foo} }"},     // mismatched delimiters
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			_, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err == nil {
				t.Fatalf("expected error but had none:\n%s", tt.src)
			}
		})
	}
}
