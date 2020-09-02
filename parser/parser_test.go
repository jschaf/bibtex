package parser

import (
	gotok "go/token"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/asts"
	"github.com/jschaf/bibtex/token"
)

func cmpExpr() cmp.Option {
	return cmp.Transformer("expr_name", func(x ast.Expr) string {
		return asts.ExprString(x)
	})
}

func cmpIdentName() cmp.Option {
	return cmp.Transformer("ident_name", func(x *ast.Ident) string {
		return x.Name
	})
}

func cmpTagEntry() cmp.Option {
	return cmp.Transformer("tag_entry", func(t *ast.TagStmt) string {
		return t.RawName + " = " + asts.ExprString(t.Value)
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
		{"@PREAMBLE { {foo} }", asts.UnparsedBraceText("foo")},
		{`@PREAMBLE { "foo" }`, asts.UnparsedText("foo")},
		{`@PREAMBLE ( "foo" )`, asts.UnparsedText("foo")},
		{`@preamble { "foo" }`, asts.UnparsedText("foo")},
		{`@preamble { "foo" # "bar" }`, asts.Concat(asts.UnparsedText("foo"), asts.UnparsedText("bar"))},
		{`@preamble { "foo" # "bar" # "qux" }`, asts.Concat(asts.UnparsedText("foo"), asts.Concat(asts.UnparsedText("bar"), asts.UnparsedText("qux")))},
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
		{"@string { key = {foo} }", token.BraceString, "key", asts.UnparsedBraceText("foo")},
		{"@string { KeY = {foo} }", token.BraceString, "KeY", asts.UnparsedBraceText("foo")},
		{`@string { KeY = "foo" }`, token.String, "KeY", asts.UnparsedText("foo")},
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

func TestParseFile_BibDecl_NoParseStrings(t *testing.T) {
	tests := []struct {
		src    string
		typeFn func(decl *ast.BibDecl)
		keysFn func(*ast.BibDecl)
		tagsFn func(*ast.BibDecl)
	}{
		{"@article { cite_key, key = {foo} }",
			asts.WithBibType("article"),
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("key", asts.UnparsedBraceText("foo"))},
		{"@article {cite_key1, key = {foo} }",
			asts.WithBibType("article"),
			asts.WithBibKeys("cite_key1"),
			asts.WithBibTags("key", asts.UnparsedBraceText("foo"))},
		{"@article {111, key = {foo} }",
			asts.WithBibType("article"),
			asts.WithBibKeys("111"),
			asts.WithBibTags("key", asts.UnparsedBraceText("foo"))},
		{"@article {111, key = bar }",
			asts.WithBibType("article"),
			asts.WithBibKeys("111"),
			asts.WithBibTags("key", asts.Ident("bar"))},
		{"@article {111, key = bar },", // trailing comma
			asts.WithBibType("article"),
			asts.WithBibKeys("111"),
			asts.WithBibTags("key", asts.Ident("bar"))},
		{"@article {111, key = bar, extra }",
			asts.WithBibType("article"),
			asts.WithBibKeys("111", "extra"),
			asts.WithBibTags("key", asts.Ident("bar"))},
		{`@inproceeding {111, key = bar, a, b, k2 = "v2" }`,
			asts.WithBibType("inproceeding"),
			asts.WithBibKeys("111", "a", "b"),
			asts.WithBibTags("key", asts.Ident("bar"), "k2", asts.UnparsedText("v2"))},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err != nil {
				t.Fatal(err)
			}

			gotBib := f.Entries[0].(*ast.BibDecl)
			wantBib := &ast.BibDecl{}
			tt.typeFn(wantBib)
			tt.keysFn(wantBib)
			tt.tagsFn(wantBib)

			if diff := cmp.Diff(wantBib.Type, gotBib.Type); diff != "" {
				t.Errorf("BibDecl type mismatch (-want +got):\n%s", diff)
			}
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
		{"@article { cite_key, key = {foo} }",
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("key", asts.BraceText(0, "foo"))},
		{"@article { cite_key, key = {{f}oo}}",
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("key", asts.BraceText(0, "{f}", "oo"))},
		{`@article { cite_key, key = {{\textsc f}oo}}`,
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("key", asts.BraceText(0, `{\textsc f}`, "oo"))},
		{`@article { cite_key, key = "foo" }`,
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("key", asts.QuotedText(0, "foo"))},
		{"@article { cite_key, key = {f\no\ro} }",
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("key", asts.BraceText(0, "f", " ", "o", " ", "o"))},
		{`@article{cite_key, howPublished = "\url{http://example.com/foo--bar/~baz/#}" }`,
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("howPublished",
				asts.QuotedTextExpr(0, asts.CmdText("url", "http://example.com/foo--bar/~baz/#")))},
		{`@article{cite_key, url = "http://example.com/foo--bar/~baz/#" }`,
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("url", asts.QuotedText(0, "http://example.com/foo--bar/~baz/#"))},
		{`@article{cite_key, url = "\url{http://foo.com/bar~qux-baz/#}" }`,
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("url",
				asts.QuotedTextExpr(0, asts.CmdText("url", "http://foo.com/bar~qux-baz/#")))},
		{`@article { cite_key, title = {\href{https://nyt.com/}{Dollar \$140}} }`,
			asts.WithBibKeys("cite_key"),
			asts.WithBibTags("title",
				asts.BraceText(0, `\href`, "{https://nyt.com/}", `{Dollar \$140}`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, ParseStrings|Trace)
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

func TestParseExpr_BibDecl_ModeParseStrings(t *testing.T) {
	tests := []struct {
		src  string
		want ast.Expr
	}{
		{"{foo}", asts.BraceText(0, "foo")},
		{`"foo"`, asts.QuotedText(0, "foo")},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got, err := ParseExpr(tt.src)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tt.want, got, cmpExpr()); diff != "" {
				t.Errorf("ParseExpr mismatch (-want +got):\n%s", diff)
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

func TestParseFile_BibDecl_invalid_ParseStrings(t *testing.T) {
	tests := []struct {
		src string
	}{
		{"@article {111, a = {$x=1} }"}, // unmatched math ($) delimiter
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			_, err := ParseFile(gotok.NewFileSet(), "", tt.src, ParseStrings)
			if err == nil {
				t.Fatalf("expected error but had none:\n%s", tt.src)
			}
		})
	}
}
