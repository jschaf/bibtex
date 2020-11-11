package bibtex

import (
	"github.com/jschaf/bibtex/asts"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/bibtex/ast"
)

func TestSimplifyEscapedTextResolver(t *testing.T) {
	tests := []struct {
		node ast.Expr
		want ast.Expr
	}{
		{
			asts.BraceText(0, asts.Escaped('&')),
			asts.BraceText(0, asts.Text("&")),
		},
		{
			asts.BraceText(0, asts.BraceText(1, "abc", asts.Escaped('&'), "def")),
			asts.BraceText(0, asts.BraceText(1, "abc", "&", "def")),
		},
	}

	for _, tt := range tests {
		t.Run(asts.ExprString(tt.node), func(t *testing.T) {
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
