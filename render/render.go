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
	textOverrides map[ast.TextKind]TextRendererFunc
}

type Option func(p *TextRenderer)

func WithTextOverride(kind ast.TextKind, r TextRendererFunc) Option {
	return func(p *TextRenderer) {
		p.textOverrides[kind] = r
	}
}

func NewTextRenderer(opts ...Option) *TextRenderer {
	p := &TextRenderer{
		textOverrides: make(map[ast.TextKind]TextRendererFunc),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
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
	case *ast.MacroText:
		for _, v := range t.Values {
			if mErr = p.Render(w, v); mErr != nil {
				return
			}
		}
	case *ast.Text:
		switch t.Kind {
		case ast.TextComma:
			if r, ok := p.textOverrides[ast.TextComma]; ok {
				mErr = r.Render(w, t)
			} else {
				_, mErr = w.Write([]byte(","))
			}
		case ast.TextContent:
			if r, ok := p.textOverrides[ast.TextContent]; ok {
				mErr = r.Render(w, t)
			} else {
				_, mErr = w.Write([]byte(t.Value))
			}
		case ast.TextEscaped:
			if r, ok := p.textOverrides[ast.TextEscaped]; ok {
				mErr = r.Render(w, t)
			} else {
				_, mErr = w.Write([]byte(t.Value))
			}
		case ast.TextHyphen:
			if r, ok := p.textOverrides[ast.TextHyphen]; ok {
				mErr = r.Render(w, t)
			} else {
				_, mErr = w.Write([]byte("-"))
			}
		case ast.TextMath:
			if r, ok := p.textOverrides[ast.TextMath]; ok {
				mErr = r.Render(w, t)
			} else {
				if _, mErr = w.Write([]byte("$")); mErr != nil {
					return mErr
				}
				if _, mErr = w.Write([]byte(t.Value)); mErr != nil {
					return mErr
				}
				if _, mErr = w.Write([]byte("$")); mErr != nil {
					return mErr
				}
			}
		case ast.TextNBSP, ast.TextSpace:
			_, mErr = w.Write([]byte(" "))
		default:
			return fmt.Errorf("renderer - unhandled ast.Text value: %s", t.Value)
		}
	default:
		return fmt.Errorf("renderer - unhandled ast.Expr type %T, %v", t, t)
	}
	return nil
}
