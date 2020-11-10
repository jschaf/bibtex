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

// Resolver is an in-place mutation of an ast.Node to support resolving Bibtex
// entries. Typically, the mutations simplify the AST to support easier
// manipulation, like replacing ast.EscapedText with the escaped value.
type Resolver interface {
	Resolve(n ast.Node) error
}

type ResolverFunc func(n ast.Node) error

func (r ResolverFunc) Resolve(n ast.Node) error {
	return r(n)
}

// SimplifyEscapedTextResolver replaces ast.TextEscaped nodes with a plain
// ast.Text containing the value that was escaped. Meaning, `\&` is converted to
// `&`.
func SimplifyEscapedTextResolver(root ast.Node) error {
	err := ast.Walk(root, func(n ast.Node, isEntering bool) (ast.WalkStatus, error) {
		if txt, ok := n.(*ast.ParsedText); ok {
			for i, val := range txt.Values {
				if esc, ok := val.(*ast.TextEscaped); ok {
					txt.Values[i] = &ast.Text{Value: esc.Value}
				}
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return fmt.Errorf("simplify escaped text resolver: %w", err)
	}
	return nil
}

// TODO: Add AuthorResolver

const authorSep = "and"

func ResolveFile(fset *gotok.FileSet, filename string, src interface{}) ([]ASTEntry, error) {
	f, err := parser.ParseFile(fset, filename, src, parser.ParseStrings)
	if err != nil {
		return nil, err
	}
	entries := make([]ASTEntry, 0, len(f.Entries))
	for _, rawE := range f.Entries {
		if _, ok := rawE.(*ast.BibDecl); !ok {
			continue
		}
		bibDecl := rawE.(*ast.BibDecl)
		normE := ASTEntry{
			Key:  bibDecl.Key.Name,
			Type: bibDecl.Type,
			Tags: make(map[Field]ast.Expr),
		}
		for _, tag := range bibDecl.Tags {
			normE.Tags[tag.Name] = tag.Value
		}
		entries = append(entries, normE)
	}
	return entries, nil
}

// ResolveAuthors extracts the authors from the parsed text of a bibtex field,
// usually from the author or editor field of bibtex entry.
func ResolveAuthors(txt *ast.ParsedText) ([]Author, error) {
	// Pop tokens until we get the author separator then hand off the tokens
	// to resolveAuthor to get the Author.
	authors := make([]Author, 0, 4)
	aExprs := make([]ast.Expr, 0, 8)
	for _, v := range txt.Values {
		switch t := v.(type) {
		case *ast.Text:
			if t.Value == authorSep {
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
	if _, ok := x.(*ast.Text); !ok {
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

// resolveAuthorN resolves an author entry that contains 1 or more commas.
//
//     1 comma:  last, first             => Author{first, "",  last, ""}
//     1 comma:  von last, first         => Author(first, von, last, "")
//     2 commas: last, suffix, first     => Author{first, "",  last, suffix}
//     2 commas: von last, suffix, first => Author{first, von, last, suffix}
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
		if _, ok := x.(*ast.TextComma); ok {
			idxs = append(idxs, i)
		}
	}
	return idxs
}
