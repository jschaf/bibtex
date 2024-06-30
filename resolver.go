package bibtex

import (
	"fmt"
	"strings"

	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/render"
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
		if !isEntering {
			return ast.WalkSkipChildren, nil
		}
		if _, ok := n.(*ast.ParsedText); !ok {
			return ast.WalkContinue, nil
		}
		txt := n.(*ast.ParsedText)
		for i, val := range txt.Values {
			if esc, ok := val.(*ast.TextEscaped); ok {
				txt.Values[i] = &ast.Text{Value: esc.Value}
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return fmt.Errorf("simplify escaped text resolver: %w", err)
	}
	return nil
}

// RenderParsedTextResolver replaces ast.ParsedText with a simplified rendering
// of ast.Text.
type RenderParsedTextResolver struct {
	rend *render.TextRenderer
}

func NewRenderParsedTextResolver() *RenderParsedTextResolver {
	return &RenderParsedTextResolver{rend: render.NewTextRenderer()}
}

func (r *RenderParsedTextResolver) Resolve(root ast.Node) error {
	err := ast.Walk(root, func(n ast.Node, isEntering bool) (ast.WalkStatus, error) {
		if !isEntering {
			return ast.WalkSkipChildren, nil
		}
		tag, ok := n.(*ast.TagStmt)
		if !ok {
			return ast.WalkContinue, nil
		}

		switch expr := tag.Value.(type) {
		case ast.Authors:
			// Descend into authors and render each author part, like first
			// name and last name.
			return ast.WalkContinue, nil

		case *ast.ParsedText:
			sb := &strings.Builder{}
			sb.Grow(16)
			if err := r.rend.Render(sb, expr); err != nil {
				return ast.WalkStop, fmt.Errorf("render parsed text tag=%q: %w", tag.Name, err)
			}
			tag.Value = &ast.Text{Value: sb.String()}
			for i, val := range expr.Values {
				if esc, ok := val.(*ast.TextEscaped); ok {
					expr.Values[i] = &ast.Text{Value: esc.Value}
				}
			}
			return ast.WalkContinue, nil

		default:
			return ast.WalkSkipChildren, nil
		}
	})
	if err != nil {
		return fmt.Errorf("simplify escaped text resolver: %w", err)
	}
	return nil
}

// AuthorResolver extracts ast.Authors from the expression value of a tag
// statement.
type AuthorResolver struct {
	tags map[string]struct{} // tag names to extract authors from
}

func NewAuthorResolver(tags ...string) AuthorResolver {
	m := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		m[tag] = struct{}{}
	}
	return AuthorResolver{tags: m}
}

func (a AuthorResolver) Resolve(root ast.Node) error {
	err := ast.Walk(root, func(n ast.Node, isEntering bool) (ast.WalkStatus, error) {
		if !isEntering {
			return ast.WalkSkipChildren, nil
		}
		if _, ok := n.(*ast.TagStmt); !ok {
			return ast.WalkContinue, nil
		}
		tag := n.(*ast.TagStmt)
		if _, ok := a.tags[tag.Name]; !ok {
			return ast.WalkContinue, nil
		}
		txt, ok := tag.Value.(*ast.ParsedText)
		if !ok {
			return ast.WalkStop, fmt.Errorf("author resolver tag %q expression was not ParsedText; got %T", tag.Name, tag.Value)
		}
		authors, err := ExtractAuthors(txt)
		if err != nil {
			return ast.WalkStop, fmt.Errorf("author resolver resolution: %w", err)
		}
		tag.Value = authors

		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		return fmt.Errorf("simplify escaped text resolver: %w", err)
	}
	return nil
}
