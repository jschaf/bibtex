package parser

import (
	"fmt"
	gotok "go/token"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/b2/pkg/bibtex/ast"
	"github.com/jschaf/b2/pkg/bibtex/token"
)

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

func TestParseFile_PreambleDecl(t *testing.T) {
	tests := []struct {
		src  string
		tok  token.Token
		want string
	}{
		{"@PREAMBLE { {foo} }", token.BraceString, "foo"},
		{`@PREAMBLE { "foo" }`, token.String, "foo"},
		{`@preamble { "foo" }`, token.String, "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err != nil {
				t.Fatal(err)
			}

			txt := f.Entries[0].(*ast.PreambleDecl).Text
			if txt.Value != tt.want {
				t.Errorf("PreambleDecl value: got %s; want %s", txt.Value, tt.want)
			}
			if txt.Kind != tt.tok {
				t.Errorf("PreambleDecl kind: got %s; want %s", txt.Kind, tt.tok)
			}
		})
	}
}

func TestParseFile_AbbrevDecl(t *testing.T) {
	tests := []struct {
		src              string
		tok              token.Token
		wantKey, wantVal string
	}{
		{"@string { key = {foo} }", token.BraceString, "key", "foo"},
		{"@string { KeY = {foo} }", token.BraceString, "KeY", "foo"},
		{`@string { KeY = "foo" }`, token.String, "KeY", "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			f, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err != nil {
				t.Fatal(err)
			}

			tag := f.Entries[0].(*ast.AbbrevDecl).Tag

			if tag.RawName != tt.wantKey {
				t.Errorf("AbbrevDecl raw key: got %s; want %s", tag.RawName, tt.wantKey)
			}
			wantNormKey := strings.ToLower(tt.wantKey)
			if tag.Name != wantNormKey {
				t.Errorf("AbbrevDecl name: got %s; want %s", tag.Name, wantNormKey)
			}
			val := ""
			kind := token.Illegal
			if t, ok := tag.Value.(*ast.BasicLit); ok {
				val = t.Value
				kind = t.Kind
			}
			if val != tt.wantVal {
				t.Errorf("AbbrevDecl value: got %s; want %s", val, tt.wantVal)
			}
			if kind != tt.tok {
				t.Errorf("AbbrevDecl value token: got %s; want %s", kind, tt.tok)
			}
		})
	}
}

func bibKeys(ts ...string) func(decl *ast.BibDecl) {
	return func(b *ast.BibDecl) {
		for _, k := range ts {
			b.Keys = append(b.Keys, &ast.Ident{Name: k})
		}
	}
}

func bibTags(ts ...string) func(decl *ast.BibDecl) {
	if len(ts)%2 != 0 {
		panic("bibTags must have even number of strings for key-val pairs")
	}
	return func(b *ast.BibDecl) {
		for i := 0; i < len(ts); i += 2 {
			key, val := ts[i], ts[i+1]
			tag := &ast.TagStmt{
				Name:    key,
				RawName: key,
				Value: &ast.BasicLit{
					Kind:  token.BraceString,
					Value: val,
				},
			}
			b.Tags = append(b.Tags, tag)
		}
	}
}

func cmpIdentName() cmp.Option {
	return cmp.Transformer("ident_name", func(x *ast.Ident) string {
		return x.Name
	})
}

func cmpTagEntry() cmp.Option {
	return cmp.Transformer("tag_entry", func(t *ast.TagStmt) string {
		k := t.RawName
		var val string
		switch v := t.Value.(type) {
		case *ast.BasicLit:
			val = v.Kind.String() + "(" + v.Value + ")"
		default:
			val = fmt.Sprintf("%v", v)
		}
		return k + " = " + val
	})
}

func TestParseFile_BibDecl(t *testing.T) {
	tests := []struct {
		src    string
		keysFn func(*ast.BibDecl)
		tagsFn func(*ast.BibDecl)
	}{
		{"@article { cite_key, key = {foo} }", bibKeys("cite_key"), bibTags("key", "foo")},
		{"@article {cite_key1, key = {foo} }", bibKeys("cite_key1"), bibTags("key", "foo")},
		{"@article {111, key = {foo} }", bibKeys("111"), bibTags("key", "foo")},
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

			if diff := cmp.Diff(wantBib.Keys, gotBib.Keys, cmpIdentName()); diff != "" {
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
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			_, err := ParseFile(gotok.NewFileSet(), "", tt.src, 0)
			if err == nil {
				t.Fatal(err)
			}
		})
	}
}
