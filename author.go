package bibtex

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/render"
)

const authorSep = "and"

// ExtractAuthors extracts the authors from the parsed text of a bibtex field,
// usually from the author or editor field of bibtex entry.
func ExtractAuthors(txt *ast.ParsedText) (ast.Authors, error) {
	// Pop tokens until we get the author separator then hand off the tokens
	// to extractAuthor to get the Author.
	authors := make([]*ast.Author, 0, 4)
	aExprs := make([]ast.Expr, 0, 8)
	for _, v := range txt.Values {
		switch t := v.(type) {
		case *ast.Text:
			if t.Value == authorSep {
				a := extractAuthor(aExprs)
				if a.IsEmpty() {
					return nil, fmt.Errorf("found an empty author")
				}
				authors = append(authors, a)
				aExprs = aExprs[:0]
				continue
			}
		}
		aExprs = append(aExprs, v)
	}
	final := extractAuthor(aExprs)
	if final.IsEmpty() {
		return nil, fmt.Errorf("found an empty author")
	}
	authors = append(authors, final)
	return authors, nil
}

func trimSpaces(xs []ast.Expr) []ast.Expr {
	lo, hi := 0, len(xs)

	for i := 0; i < len(xs); i++ {
		if _, ok := xs[i].(*ast.TextSpace); !ok {
			break
		}
		lo++
	}

	for i := len(xs) - 1; i >= 0; i-- {
		if _, ok := xs[i].(*ast.TextSpace); !ok {
			break
		}
		hi--
	}
	if hi <= lo {
		return xs[lo:lo]
	}
	return xs[lo:hi]
}

func extractAuthor(xs []ast.Expr) *ast.Author {
	xs = trimSpaces(xs)
	commas := findCommas(xs)
	if len(commas) == 0 {
		return resolveAuthor0(xs)
	} else {
		return resolveAuthorN(xs, commas)
	}
}

type resolveAction int

const (
	resolveContinue resolveAction = iota
	resolveNextPart
)

func parseFirstName(idx int, xs []ast.Expr) (string, resolveAction) {
	x := xs[idx]
	if _, ok := x.(*ast.TextSpace); ok && idx < len(xs)-1 {
		if t, ok := xs[idx+1].(*ast.Text); ok && !hasUpperPrefix(t.Value) {
			// lowercase after space means we're out of the first name
			return "", resolveNextPart
		}
	}

	if _, ok := x.(*ast.Text); !ok {
		return parseDefault(idx, xs), resolveContinue
	}

	t := x.(*ast.Text)

	return t.Value, resolveContinue
}

func hasUpperPrefix(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

func parsePrefixName(idx int, xs []ast.Expr) (string, resolveAction) {
	x := xs[idx]
	if _, ok := x.(*ast.Text); !ok {
		return parseDefault(idx, xs), resolveContinue
	}

	t := x.(*ast.Text)
	if hasUpperPrefix(t.Value) {
		return "", resolveNextPart // Only lowercase goes in prefix
	}
	return t.Value, resolveContinue
}

func parseLastName(idx int, xs []ast.Expr) (string, resolveAction) {
	return parseDefault(idx, xs), resolveContinue
}

// TODO: replace with single resolver
func parseDefault(idx int, xs []ast.Expr) string {
	x := xs[idx]
	switch t := x.(type) {
	case *ast.ParsedText:
		sb := strings.Builder{}
		sb.Grow(16)
		for i := range t.Values {
			d := parseDefault(i, t.Values)
			sb.WriteString(d)
		}
		return sb.String()
	case *ast.TextComma:
		return ","
	case *ast.TextEscaped:
		return `\` + t.Value
	case *ast.TextHyphen:
		return "-"
	case *ast.TextNBSP, *ast.TextSpace:
		return " "
	case *ast.TextMath:
		return "$" + t.Value + "$"
	case *ast.Text:
		return t.Value
	case *ast.TextAccent:
		r, err := render.RenderAccent(t.Accent, t.Text.Value)
		if err != nil {
			panic("cannot render accent")
		}
		return string(r)
	default:
		panic("unhandled ast.Expr value")
	}
}

// resolveAuthor0 resolves an author for an entry with no commas, like
// "First von Last".
func resolveAuthor0(xs []ast.Expr) *ast.Author {
	first := strings.Builder{}
	first.Grow(16)
	idx := 0
	for ; idx < len(xs); idx++ {
		if idx == len(xs)-1 {
			// If we're on the last part, it belongs to the last name.
			break
		}
		val, action := parseFirstName(idx, xs)
		if action == resolveNextPart {
			break
		}
		first.WriteString(val)
	}

	prefix := strings.Builder{}
	for ; idx < len(xs); idx++ {
		if idx == len(xs)-1 {
			// If we're on the last part, it belongs to the last name.
			break
		}
		val, action := parsePrefixName(idx, xs)
		if action == resolveNextPart {
			break
		}
		prefix.WriteString(val)
	}

	last := strings.Builder{}
	last.Grow(16)
	for ; idx < len(xs); idx++ {
		val, action := parseLastName(idx, xs)
		if action == resolveNextPart {
			break
		}
		last.WriteString(val)
	}

	return &ast.Author{
		First:  &ast.Text{Value: strings.TrimSpace(first.String())},
		Prefix: &ast.Text{Value: strings.TrimSpace(prefix.String())},
		Last:   &ast.Text{Value: strings.TrimSpace(last.String())},
		Suffix: &ast.Text{Value: ""},
	}
}

// resolveAuthorN resolves an author entry that contains 1 or more commas.
//
//	1 comma:  last, first             => Author{first, "",  last, ""}
//	1 comma:  von last, first         => Author(first, von, last, "")
//	2 commas: last, suffix, first     => Author{first, "",  last, suffix}
//	2 commas: von last, suffix, first => Author{first, von, last, suffix}
func resolveAuthorN(xs []ast.Expr, commas []int) *ast.Author {
	part1 := xs[:commas[0]]
	idx1 := 0
	prefix := strings.Builder{}
	for ; idx1 < len(part1); idx1++ {
		if idx1 == len(part1)-1 {
			// If we're on the last part, it belongs to the last name.
			break
		}
		val, action := parsePrefixName(idx1, part1)
		if action == resolveNextPart {
			break
		}
		prefix.WriteString(val)
	}

	last := strings.Builder{}
	last.Grow(16)
	for ; idx1 < len(part1); idx1++ {
		val, action := parseLastName(idx1, part1)
		if action == resolveNextPart {
			break
		}
		last.WriteString(val)
	}

	part2 := xs[commas[0]+1:]

	suffix := strings.Builder{}
	suffix.Grow(16)
	if len(commas) > 1 {
		val := parseDefault(0, xs[commas[0]+1:])
		suffix.WriteString(val)
		part2 = xs[commas[1]+1:]
	}

	idx2 := 0
	first := strings.Builder{}
	first.Grow(16)
	for ; idx2 < len(part2); idx2++ {
		val, action := parseFirstName(idx2, part2)
		if action == resolveNextPart {
			break
		}
		first.WriteString(val)
	}

	return &ast.Author{
		First:  &ast.Text{Value: strings.TrimSpace(first.String())},
		Prefix: &ast.Text{Value: strings.TrimSpace(prefix.String())},
		Last:   &ast.Text{Value: strings.TrimSpace(last.String())},
		Suffix: &ast.Text{Value: strings.TrimSpace(suffix.String())},
	}
}

// findCommas returns the offsets of all commas in xs. Only searches
// 1 level deep.
func findCommas(xs []ast.Expr) []int {
	idxs := make([]int, 0, 2)
	for i, x := range xs {
		if _, ok := x.(*ast.TextComma); ok {
			idxs = append(idxs, i)
		}
	}
	return idxs
}
