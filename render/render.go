package render

import (
	"fmt"
	"github.com/jschaf/bibtex/ast"
	"io"
)

type ExprRenderer interface {
	Render(w io.Writer, x ast.Expr) error
}

type TextRendererFunc func(w io.Writer, text *ast.Text) error

func (t TextRendererFunc) Render(w io.Writer, x ast.Expr) error {
	return t(w, x.(*ast.Text))
}

type TextRenderer struct {
}

type Option func(p *TextRenderer)

func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

func (p TextRenderer) Render(w io.Writer, x ast.Expr) (mErr error) {
	switch t := x.(type) {
	case *ast.ParsedText:
		for _, value := range t.Values {
			if mErr = p.Render(w, value); mErr != nil {
				return
			}
		}
	case *ast.ConcatExpr:
		if mErr = p.Render(w, t.X); mErr != nil {
			return
		}
		_, _ = w.Write([]byte{'#'})
		if mErr = p.Render(w, t.Y); mErr != nil {
			return
		}
	case *ast.TextMacro:
		for _, v := range t.Values {
			if mErr = p.Render(w, v); mErr != nil {
				return
			}
		}
		// TODO: add overrides and TextMacro
	case *ast.TextComma:
		_, mErr = w.Write([]byte(","))
	case *ast.Text:
		_, mErr = w.Write([]byte(t.Value))
	case *ast.TextEscaped:
		_, mErr = w.Write([]byte(t.Value))
	case *ast.TextHyphen:
		_, mErr = w.Write([]byte("-"))
	case *ast.TextMath:
		if _, mErr = w.Write([]byte("$")); mErr != nil {
			return mErr
		}
		if _, mErr = w.Write([]byte(t.Value)); mErr != nil {
			return mErr
		}
		if _, mErr = w.Write([]byte("$")); mErr != nil {
			return mErr
		}
	case *ast.TextNBSP, *ast.TextSpace:
		_, mErr = w.Write([]byte(" "))
	default:
		return fmt.Errorf("renderer - unhandled ast.Expr type %T, %v", t, t)
	}
	return nil
}
