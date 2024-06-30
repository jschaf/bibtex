package bibtex

import (
	"testing"

	"github.com/jschaf/bibtex/asts"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/bibtex/ast"
)

func TestSimplifyEscapedTextResolver(t *testing.T) {
	tests := []struct {
		name string
		node ast.Expr
		want ast.Expr
	}{
		{
			name: "escaped ampersand",
			node: asts.BraceText(0, asts.Escaped('&')),
			want: asts.BraceText(0, asts.Text("&")),
		},
		{
			name: "escaped ampersand in text",
			node: asts.BraceText(0, asts.BraceText(1, "abc", asts.Escaped('&'), "def")),
			want: asts.BraceText(0, asts.BraceText(1, "abc", "&", "def")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SimplifyEscapedTextResolver(tt.node); err != nil {
				t.Fatal(err)
			}
			want := asts.ExprString(tt.want)
			got := asts.ExprString(tt.node)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("SimplifyEscapedTextResolver() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderParsedTextResolver_Resolve(t *testing.T) {
	tests := []struct {
		name string
		node ast.Expr
		want ast.Expr
	}{
		{
			name: "escaped ampersand",
			node: asts.BraceText(0, asts.Escaped('&')),
			want: asts.BraceText(0, asts.Text("&")),
		},
		{
			name: "escaped ampersand in text",
			node: asts.BraceText(0, asts.BraceText(1, "abc", asts.Escaped('&'), "def")),
			want: asts.BraceText(0, asts.BraceText(1, "abc", "&", "def")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First run escape resolver to simplify the text.
			if err := SimplifyEscapedTextResolver(tt.node); err != nil {
				t.Fatal(err)
			}

			r := NewRenderParsedTextResolver()
			if err := r.Resolve(tt.node); err != nil {
				t.Fatal(err)
			}
			want := asts.ExprString(tt.want)
			got := asts.ExprString(tt.node)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("SimplifyEscapedTextResolver() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
