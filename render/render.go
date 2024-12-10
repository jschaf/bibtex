package render

import (
	"fmt"
	"io"
	"strings"
	"unicode"
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
			if mErr = p.Render(w, v); mErr != nil {
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
			return
		}
		if _, mErr = w.Write([]byte(t.Value)); mErr != nil {
			return
		}
		_, mErr = w.Write([]byte("$"))
	case *ast.TextNBSP, *ast.TextSpace:
		_, mErr = w.Write([]byte(" "))
	case *ast.TextAccent:
		var (
			accent       accentType
			accentedRune rune
		)
		if accent, mErr = newAccentType(t.Accent); mErr != nil {
			return
		}
		text := t.Value.(*ast.Text).Value
		if utf8.RuneCountInString(text) != 1 {
			return fmt.Errorf("renderer - accent text must be a single character")
		}
		r, _ := utf8.DecodeRuneInString(text)
		if accentedRune, mErr = fmtAccent(r, accent); mErr != nil {
			return
		}
		_, mErr = w.Write([]byte(string(accentedRune)))
	default:
		mErr = fmt.Errorf("renderer - unhandled ast.Expr type %T, %v", t, t)
	}
	return
}

// accentType represents the different LaTeX accents
type accentType string

const (
	Grave      accentType = "grave"      // \`
	Acute      accentType = "acute"      // \'
	Circumflex accentType = "circumflex" // \^
	Umlaut     accentType = "umlaut"     // \"
	Tilde      accentType = "tilde"      // \~
	Cedilla    accentType = "cedilla"    // \c
	Dot        accentType = "dot"        // \.
)

// NewAccentType converts a string representation to an AccentType
// It returns the AccentType and an error if the input is invalid
func newAccentType(accentStr string) (accentType, error) {
	// Normalize the input string to lowercase
	accentStr = strings.ToLower(strings.TrimSpace(accentStr))

	// Map of string representations to AccentType
	accentMap := map[string]accentType{
		// LaTeX command mappings
		"`":  Grave,
		"'":  Acute,
		"^":  Circumflex,
		"\"": Umlaut,
		"~":  Tilde,
		"c":  Cedilla,
		".":  Dot,
	}

	// Look up the accent type
	if accentType, exists := accentMap[accentStr]; exists {
		return accentType, nil
	}

	// Generate a helpful error message with available options
	availableAccents := make([]string, 0, len(accentMap))
	for k := range accentMap {
		// Exclude LaTeX command symbols to avoid cluttering the error message
		if len(k) > 1 {
			availableAccents = append(availableAccents, k)
		}
	}

	return "", fmt.Errorf("invalid accent type. Available accent types are: %v",
		strings.Join(availableAccents, ", "))
}

// ConvertLatexAccent converts a base character with a LaTeX accent to its UTF-8 equivalent
func fmtAccent(baseChar rune, accent accentType) (rune, error) {
	// Normalize the base character to lowercase for consistent mapping
	baseChar = unicode.ToLower(baseChar)

	// Mapping of base characters to their accented versions
	accentMap := map[accentType]map[rune]rune{
		Grave: {
			'a': 'à', 'e': 'è', 'i': 'ì', 'o': 'ò', 'u': 'ù',
			'A': 'À', 'E': 'È', 'I': 'Ì', 'O': 'Ò', 'U': 'Ù',
		},
		Acute: {
			'a': 'á', 'e': 'é', 'i': 'í', 'o': 'ó', 'u': 'ú', 'y': 'ý',
			'A': 'Á', 'E': 'É', 'I': 'Í', 'O': 'Ó', 'U': 'Ú', 'Y': 'Ý',
		},
		Circumflex: {
			'a': 'â', 'e': 'ê', 'i': 'î', 'o': 'ô', 'u': 'û',
			'A': 'Â', 'E': 'Ê', 'I': 'Î', 'O': 'Ô', 'U': 'Û',
		},
		Umlaut: {
			'a': 'ä', 'e': 'ë', 'i': 'ï', 'o': 'ö', 'u': 'ü',
			'A': 'Ä', 'E': 'Ë', 'I': 'Ï', 'O': 'Ö', 'U': 'Ü',
		},
		Tilde: {
			'a': 'ã', 'n': 'ñ', 'o': 'õ',
			'A': 'Ã', 'N': 'Ñ', 'O': 'Õ',
		},
		Cedilla: {
			'c': 'ç',
			'C': 'Ç',
		},
		Dot: {
			'c': 'ċ', 'e': 'ė', 'g': 'ġ', 'i': 'ı', 'z': 'ż',
			'C': 'Ċ', 'E': 'Ė', 'G': 'Ġ', 'I': 'İ', 'Z': 'Ż',
		},
	}

	// Check if the accent exists for the given base character
	if accentedChars, exists := accentMap[accent]; exists {
		if accented, valid := accentedChars[baseChar]; valid {
			return accented, nil
		}
	}

	// Generate a helpful error message
	availableChars := []rune{}
	for char := range accentMap[accent] {
		availableChars = append(availableChars, char)
	}

	return 0, fmt.Errorf("invalid combination: cannot apply %s accent to character '%c'. "+
		"Available characters for this accent are: %v",
		accent, baseChar, string(availableChars))
}
