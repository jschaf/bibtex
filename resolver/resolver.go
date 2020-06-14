// Package resolver transforms a Bibtex AST into complete bibtex entries,
// resolving cross references, parsing authors and editors, and normalizing
// page numbers.
package resolver

import (
	"github.com/jschaf/b2/pkg/bibtex"
	"github.com/jschaf/b2/pkg/bibtex/ast"
	"strings"
	"unicode"
	"unicode/utf8"
)

const authorSep = "and"

// ResolveAuthors extracts the authors from the parsed text of a bibtex field,
// usually the author or editor field.
func ResolveAuthors(txt *ast.ParsedText) ([]bibtex.Author, error) {
	// Pop tokens until we get the author separator.
	//   keep track of the number of commas at brace level 1.
	//   hand off tokens to resolver
	authors := make([]bibtex.Author, 0, 4)
	aExprs := make([]ast.Expr, 0, 8)
	for _, v := range txt.Values {
		switch t := v.(type) {
		case *ast.Text:
			if t.Kind == ast.TextContent && t.Value == authorSep {
				a := resolveAuthor(aExprs)
				authors = append(authors, a)
				aExprs = aExprs[:0]
				continue
			}
		}
		aExprs = append(aExprs, v)
	}
	final := resolveAuthor(aExprs)
	authors = append(authors, final)
	return authors, nil
}

func trimSpaces(xs []ast.Expr) []ast.Expr {
	lo, hi := 0, len(xs)
	for i := 0; i < len(xs); i++ {
		if t, ok := xs[i].(*ast.Text); !ok || t.Kind != ast.TextSpace {
			break
		}
		lo++
	}

	for i := len(xs) - 1; i >= 0; i-- {
		if t, ok := xs[i].(*ast.Text); !ok || t.Kind != ast.TextSpace {
			break
		}
		hi--
	}
	return xs[lo:hi]
}

func resolveAuthor(xs []ast.Expr) bibtex.Author {
	xs = trimSpaces(xs)
	commas := findCommas(xs)
	switch len(commas) {
	case 0:
		return resolveAuthor0(xs)
	case 1:
		return resolveAuthor1(xs, commas)
	}
	panic("unreachable")
}

type resolveAction int

const (
	resolveContinue resolveAction = iota
	resolveNextPart
)

func parseFirstName(idx int, xs []ast.Expr) (string, resolveAction) {
	x := xs[idx]
	if t, ok := x.(*ast.Text); !ok || t.Kind != ast.TextContent {
		return parseDefault(idx, xs), resolveContinue
	}

	t := x.(*ast.Text)
	if !hasUpperPrefix(t.Value) {
		return "", resolveNextPart // lowercase means we're out of the first name
	}

	return t.Value, resolveContinue
}

func hasUpperPrefix(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

func parsePrefixName(idx int, xs []ast.Expr) (string, resolveAction) {
	x := xs[idx]
	if t, ok := x.(*ast.Text); !ok || t.Kind != ast.TextContent {
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
	case *ast.Text:
		switch t.Kind {
		case ast.TextComma:
			panic("unexpected comma")
		case ast.TextNBSP, ast.TextSpace:
			return " "
		case ast.TextContent:
			return t.Value
		case ast.TextHyphen:
			return "-"
		case ast.TextMath:
			return "$" + t.Value + "$"
		case ast.TextSpecial:
			// TODO: handle accents
			return t.Value
		default:
			panic("unhandled ast.Text value: " + t.Value)
		}
	default:
		panic("unhandled ast.Expr value")
	}
}

// resolveAuthor0 resolves an author for an entry with no commas, like
// "First von Last".
func resolveAuthor0(xs []ast.Expr) bibtex.Author {
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

	return bibtex.Author{
		First:  strings.TrimRight(first.String(), " "),
		Prefix: strings.TrimRight(prefix.String(), " "),
		Last:   strings.TrimRight(last.String(), " "),
		Suffix: "",
	}
}

// resolveAuthor1 resolves an author for an entry with one comma, like
// "von Last, First".
func resolveAuthor1(xs []ast.Expr, commas []int) bibtex.Author {
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

	return bibtex.Author{
		First:  strings.Trim(first.String(), " "),
		Prefix: strings.Trim(prefix.String(), " "),
		Last:   strings.Trim(last.String(), " "),
		Suffix: "",
	}
}

// findCommas returns the offsets of all commas in xs. Only searches
// 1 level deep.
func findCommas(xs []ast.Expr) []int {
	idxs := make([]int, 0, 2)
	for i, x := range xs {
		if t, ok := x.(*ast.Text); ok && t.Kind == ast.TextComma {
			idxs = append(idxs, i)
		}
	}
	return idxs
}
