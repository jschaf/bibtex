// Package resolver transforms a Bibtex AST into complete bibtex entries,
// resolving cross references, parsing authors and editors, and normalizing
// page numbers.
package bibtex

import (
	"fmt"
	gotok "go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/parser"
)

const authorSep = "and"

func exprText(x ast.Expr) string {
	switch t := x.(type) {
	case *ast.BadExpr:
		return "<bad_expr>"
	case *ast.Ident:
		return "<unresolved ident '" + t.Name + "'>"
	case *ast.Number:
		return t.Value
	case *ast.UnparsedText:
		return t.Value
	case *ast.ParsedText:
		sb := strings.Builder{}
		sb.Grow(16)
		for _, v := range t.Values {
			d := exprText(v)
			sb.WriteString(d)
		}
		return sb.String()
	case *ast.Text:
		switch t.Kind {
		case ast.TextComma:
			return ","
		case ast.TextContent:
			return t.Value
		case ast.TextEscaped:
			return t.Value
		case ast.TextHyphen:
			return "-"
		case ast.TextMath:
			return "$" + t.Value + "$"
		case ast.TextNBSP:
			return " "
		case ast.TextSpace:
			return " "
		default:
			panic("unhandled ast.Text value: " + t.Value)
		}
	case *ast.ConcatExpr:
		left := exprText(t.X)
		right := exprText(t.Y)
		return left + right
	default:
		panic("unhandled ast.Expr value")
	}
}

func ResolveFile(fset *gotok.FileSet, filename string, src interface{}) ([]Entry, error) {
	f, err := parser.ParseFile(fset, filename, src, parser.ParseStrings)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(f.Entries))
	for _, rawE := range f.Entries {
		if _, ok := rawE.(*ast.BibDecl); !ok {
			continue
		}
		bibDecl := rawE.(*ast.BibDecl)
		normE := Entry{
			Key:  bibDecl.Key.Name,
			Type: bibDecl.Type,
			Tags: make(map[Field]string),
		}
		for _, tag := range bibDecl.Tags {
			switch tag.Name {
			case FieldAuthor:
				if as, ok := tag.Value.(*ast.ParsedText); ok {
					normE.Author, err = ResolveAuthors(as)
					if err != nil {
						return nil, fmt.Errorf("resolve authors in key %v: %w", bibDecl.Key.Name, err)
					}
				}

			case FieldEditor:
				if as, ok := tag.Value.(*ast.ParsedText); ok {
					normE.Editor, err = ResolveAuthors(as)
					if err != nil {
						return nil, fmt.Errorf("resolve authors in key %v: %w", bibDecl.Key.Name, err)
					}
				}
			default:
				normE.Tags[tag.Name] = exprText(tag.Value)
			}
		}
		entries = append(entries, normE)
	}
	return entries, nil
}

// ResolveAuthors extracts the authors from the parsed text of a bibtex field,
// usually the author or editor field.
func ResolveAuthors(txt *ast.ParsedText) ([]Author, error) {
	// Pop tokens until we get the author separator.
	//   keep track of the number of commas at brace level 1.
	//   hand off tokens to resolver
	authors := make([]Author, 0, 4)
	aExprs := make([]ast.Expr, 0, 8)
	for _, v := range txt.Values {
		switch t := v.(type) {
		case *ast.Text:
			if t.Kind == ast.TextContent && t.Value == authorSep {
				a := resolveAuthor(aExprs)
				if (a == Author{}) {
					return authors, fmt.Errorf("found an empty author")
				}
				authors = append(authors, a)
				aExprs = aExprs[:0]
				continue
			}
		}
		aExprs = append(aExprs, v)
	}
	final := resolveAuthor(aExprs)
	if (final == Author{}) {
		return authors, fmt.Errorf("found an empty author")
	}
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
	if hi <= lo {
		return xs[lo:lo]
	}
	return xs[lo:hi]
}

func resolveAuthor(xs []ast.Expr) Author {
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
			return ","
		case ast.TextNBSP, ast.TextSpace:
			return " "
		case ast.TextContent:
			return t.Value
		case ast.TextHyphen:
			return "-"
		case ast.TextMath:
			return "$" + t.Value + "$"
		default:
			panic("unhandled ast.Text value: " + t.Value)
		}
	default:
		panic("unhandled ast.Expr value")
	}
}

// resolveAuthor0 resolves an author for an entry with no commas, like
// "First von Last".
func resolveAuthor0(xs []ast.Expr) Author {
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

	return Author{
		First:  strings.TrimRight(first.String(), " "),
		Prefix: strings.TrimRight(prefix.String(), " "),
		Last:   strings.TrimRight(last.String(), " "),
		Suffix: "",
	}
}

// resolveAuthorN resolves an author in the presense of 1 or more commas.
// last, first
// von last, first -> author(first, von, last)
// last, suffix, first -> Author{ first, "", last, suffix }
// von last, suffix, first -> Author{ first, von, last, suffix }
func resolveAuthorN(xs []ast.Expr, commas []int) Author {
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

	return Author{
		First:  strings.TrimSpace(first.String()),
		Prefix: strings.TrimSpace(prefix.String()),
		Last:   strings.TrimSpace(last.String()),
		Suffix: strings.TrimSpace(suffix.String()),
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
