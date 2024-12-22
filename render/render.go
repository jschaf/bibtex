package render

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/jschaf/bibtex/ast"
)

type NodeRenderer interface {
	Render(w io.Writer, n ast.Node, entering bool) (ast.WalkStatus, error)
}

type NodeRendererFunc func(w io.Writer, n ast.Node, entering bool) (ast.WalkStatus, error)

func (nf NodeRendererFunc) Render(w io.Writer, n ast.Node, entering bool) (ast.WalkStatus, error) {
	return nf(w, n, entering)
}

func Defaults() []NodeRenderer {
	return []NodeRenderer{
		ast.KindTexComment:      NodeRendererFunc(renderTexComment),
		ast.KindTexCommentGroup: NodeRendererFunc(renderTexCommentGroup),
		ast.KindBadExpr:         NodeRendererFunc(renderBadExpr),
		ast.KindIdent:           NodeRendererFunc(renderIdent),
		ast.KindNumber:          NodeRendererFunc(renderNumber),
		ast.KindAuthors:         NodeRendererFunc(renderAuthors),
		ast.KindAuthor:          NodeRendererFunc(renderAuthor),
		ast.KindUnparsedText:    NodeRendererFunc(renderUnparsedText),
		ast.KindParsedText:      NodeRendererFunc(renderParsedText),
		ast.KindText:            NodeRendererFunc(renderText),
		ast.KindTextComma:       NodeRendererFunc(renderTextComma),
		ast.KindTextEscaped:     NodeRendererFunc(renderTextEscaped),
		ast.KindTextHyphen:      NodeRendererFunc(renderTextHyphen),
		ast.KindTextMath:        NodeRendererFunc(renderTextMath),
		ast.KindTextNBSP:        NodeRendererFunc(renderTextNBSP),
		ast.KindTextSpace:       NodeRendererFunc(renderTextSpace),
		ast.KindTextMacro:       NodeRendererFunc(renderTextMacro),
		ast.KindConcatExpr:      NodeRendererFunc(renderConcatExpr),
		ast.KindBadStmt:         NodeRendererFunc(renderBadStmt),
		ast.KindTagStmt:         NodeRendererFunc(renderTagStmt),
		ast.KindBadDecl:         NodeRendererFunc(renderBadDecl),
		ast.KindAbbrevDecl:      NodeRendererFunc(renderAbbrevDecl),
		ast.KindBibDecl:         NodeRendererFunc(renderBibDecl),
		ast.KindPreambleDecl:    NodeRendererFunc(renderPreambleDecl),
		ast.KindFile:            NodeRendererFunc(renderFile),
		ast.KindPackage:         NodeRendererFunc(renderPackage),
	}
}

func renderTexComment(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderTexCommentGroup(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderBadExpr(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkStop, fmt.Errorf("render bad expr")
}

func renderIdent(w io.Writer, n ast.Node, _ bool) (ast.WalkStatus, error) {
	ident := n.(*ast.Ident)
	_, _ = w.Write([]byte(ident.Name))
	return ast.WalkContinue, nil
}

func renderNumber(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderAuthors(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderAuthor(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderUnparsedText(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderParsedText(w io.Writer, n ast.Node, entering bool) (ast.WalkStatus, error) {
	txt := n.(*ast.ParsedText)
	left, right := `{`, `}`
	if txt.Delim == ast.QuoteDelimiter {
		left, right = `"`, `"`
	}
	if entering {
		if _, err := w.Write([]byte(left)); err != nil {
			return ast.WalkStop, fmt.Errorf("default renderParsedText left: %w", err)
		}
	} else {
		if _, err := w.Write([]byte(right)); err != nil {
			return ast.WalkStop, fmt.Errorf("default renderParsedText right: %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func renderText(w io.Writer, n ast.Node, _ bool) (ast.WalkStatus, error) {
	txt := n.(*ast.TextSpace)
	if _, err := w.Write([]byte(txt.Value)); err != nil {
		return ast.WalkStop, fmt.Errorf("default renderText: %w", err)
	}
	return ast.WalkContinue, nil
}

func renderTextComma(w io.Writer, _ ast.Node, _ bool) (ast.WalkStatus, error) {
	if _, err := w.Write([]byte(",")); err != nil {
		return ast.WalkStop, fmt.Errorf("default renderTextComma: %w", err)
	}
	return ast.WalkContinue, nil
}

func renderTextEscaped(w io.Writer, n ast.Node, _ bool) (ast.WalkStatus, error) {
	esc := n.(*ast.TextEscaped)
	if _, err := w.Write([]byte(`\` + esc.Value)); err != nil {
		return ast.WalkStop, fmt.Errorf("default renderTextEscaped: %w", err)
	}
	return ast.WalkContinue, nil
}

func renderTextHyphen(w io.Writer, _ ast.Node, _ bool) (ast.WalkStatus, error) {
	if _, err := w.Write([]byte("-")); err != nil {
		return ast.WalkStop, fmt.Errorf("default renderTextHyphen: %w", err)
	}
	return ast.WalkContinue, nil
}

func renderTextMath(w io.Writer, _ ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		if _, err := w.Write([]byte("$")); err != nil {
			return ast.WalkStop, fmt.Errorf("default renderTextMath left: %w", err)
		}
	} else {
		if _, err := w.Write([]byte("$")); err != nil {
			return ast.WalkStop, fmt.Errorf("default renderTextMath right: %w", err)
		}
	}
	return ast.WalkContinue, nil
}

func renderTextNBSP(w io.Writer, _ ast.Node, _ bool) (ast.WalkStatus, error) {
	if _, err := w.Write([]byte(" ")); err != nil {
		return ast.WalkStop, fmt.Errorf("default renderTextNBSP: %w", err)
	}
	return ast.WalkContinue, nil
}

func renderTextSpace(w io.Writer, n ast.Node, _ bool) (ast.WalkStatus, error) {
	sp := n.(*ast.TextSpace)
	if _, err := w.Write([]byte(sp.Value)); err != nil {
		return ast.WalkStop, fmt.Errorf("default renderTextSpace: %w", err)
	}
	return ast.WalkContinue, nil
}

func renderTextMacro(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	// Skip the command and write the args.
	return ast.WalkContinue, nil
}

func renderConcatExpr(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil // skip
}

func renderBadStmt(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkStop, fmt.Errorf("render bad stmt")
}

func renderTagStmt(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderBadDecl(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderAbbrevDecl(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderBibDecl(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderPreambleDecl(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderFile(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func renderPackage(io.Writer, ast.Node, bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

type TextRenderer struct{}

type Option func(p *TextRenderer)

func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

func (p TextRenderer) Render(w io.Writer, x ast.Expr) (mErr error) {
	switch t := x.(type) {
	case *ast.ParsedText:
		for _, v := range t.Values {
			if err := p.Render(w, v); err != nil {
				return err
			}
		}
	case *ast.ConcatExpr:
		if err := p.Render(w, t.X); err != nil {
			return err
		}
		_, _ = w.Write([]byte{'#'})
		if err := p.Render(w, t.Y); err != nil {
			return err
		}
	case *ast.TextMacro:
		for _, v := range t.Values {
			if err := p.Render(w, v); err != nil {
				return err
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
		if _, err := w.Write([]byte("$")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(t.Value)); err != nil {
			return err
		}
		_, mErr = w.Write([]byte("$"))
	case *ast.TextNBSP, *ast.TextSpace:
		_, mErr = w.Write([]byte(" "))
	case *ast.TextAccent:
		accent, err := NewAccentType(t.Accent)
		if err != nil {
			return err
		}
		text := t.Value.(*ast.Text).Value
		r, width := utf8.DecodeRuneInString(text)
		if r == utf8.RuneError {
			if width == 0 {
				return fmt.Errorf("empty accented text")
			}
			return fmt.Errorf("invalid UTF-8 encoding in accented text")
		}
		var remainingText = text[width:]
		accentedRune, err := FmtAccent(r, accent)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(string(accentedRune))); err != nil {
			return err
		}
		_, mErr = w.Write([]byte(remainingText))

	default:
		mErr = fmt.Errorf("renderer - unhandled ast.Expr type %T, %v", t, t)
	}
	return
}
