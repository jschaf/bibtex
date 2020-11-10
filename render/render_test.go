package render

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/asts"
	"io"
	"testing"
)

func TestTextRenderer_Render(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.Expr
		renderer *TextRenderer
		want     string
	}{
		{"simple", asts.BraceText(0, "foo"), NewTextRenderer(), "foo"},
		{"nested", asts.BraceText(0, "{foo}"), NewTextRenderer(), "foo"},
		{"text override",
			asts.BraceText(0, "{foo}"),
			NewTextRenderer(WithTextOverride(ast.TextContent, func(w io.Writer, t *ast.Text) error {
				_, _ = w.Write([]byte("bar"))
				return nil
			})),
			"bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			if err := tt.renderer.Render(buf, tt.expr); err != nil {
				t.Fatal(err)
			}
			got := buf.String()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Render() mismatch (-want +got)\n%s", diff)
			}
		})
	}
}
