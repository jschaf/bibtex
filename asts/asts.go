// Packages asts contains utilities for constructing and manipulating ASTs.
package asts

import (
	"fmt"
	"github.com/jschaf/b2/pkg/bibtex/ast"
	"github.com/jschaf/b2/pkg/bibtex/token"
	"strconv"
	"strings"
)

func UnparsedBraceText(s string) *ast.UnparsedText {
	return &ast.UnparsedText{
		Kind:  token.BraceString,
		Value: s,
	}
}

// BraceTextExpr return parsed text delimited by braces.
func BraceTextExpr(depth int, ss ...ast.Expr) *ast.ParsedText {
	return &ast.ParsedText{
		Depth:  depth,
		Delim:  ast.BraceDelimiter,
		Values: ss,
	}
}

// BraceTextExpr returns parsed text delimited by braces.
// Uses the following strategies to convert each string into a text expression:
// - If the string is all whitespace, convert to ast.TextSpace.
// - If the string begins and ends with '$', convert to ast.TextMath.
// - If the string begins with '{' and ends with '}', convert to brace text
//   recursively by removing the braces and splitting on space.
// - If the string is ',', convert to ast.TextComma.
// - Otherwise, convert to ast.Text.
func BraceText(depth int, ss ...string) *ast.ParsedText {
	xs := make([]ast.Expr, len(ss))
	for i, s := range ss {
		xs[i] = ParseStringExpr(depth, s)
	}
	return &ast.ParsedText{
		Depth:  depth,
		Delim:  ast.BraceDelimiter,
		Values: xs,
	}
}

func ParseStringExpr(depth int, s string) ast.Expr {
	switch {
	case strings.TrimSpace(s) == "":
		return WSpace()
	case strings.HasPrefix(s, "$") && strings.HasSuffix(s, "$"):
		bs := []byte(s)
		return Math(string(bs[1 : len(bs)-1]))
	case s == "~":
		return NBSP()
	case strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}"):
		bs := []byte(s)
		innerString := string(bs[1 : len(bs)-1])
		split := strings.Split(innerString, " ")
		xs := make([]ast.Expr, len(split)*2-1)
		idx := 0
		for i, sp := range split {
			innerExpr := ParseStringExpr(depth+1, sp)
			xs[idx] = innerExpr
			idx++
			if i < len(split)-1 {
				xs[idx] = WSpace()
				idx++
			}
		}
		return BraceTextExpr(depth+1, xs...)
	case s == ",":
		return Comma()
	default:
		return Text(s)
	}
}

// QuotedTextExpr return parsed text delimited by quotes.
func QuotedTextExpr(depth int, ss ...ast.Expr) *ast.ParsedText {
	return &ast.ParsedText{
		Depth:  depth,
		Delim:  ast.QuoteDelimiter,
		Values: ss,
	}
}

// QuotedText returns parsed text delimited by braces.
// Uses the following strategies to convert each string into a text expression:
// - If the string is all whitespace, convert to ast.TextSpace.
// - If the string begins and ends with '$', convert to ast.TextMath.
// - If the string begins with '{' and ends with '}', convert to brace text
//   recursively by removing the braces and splitting on space.
// - If the string is ',', convert to ast.TextComma.
// - Otherwise, convert to ast.Text.
func QuotedText(depth int, ss ...string) *ast.ParsedText {
	xs := make([]ast.Expr, len(ss))
	for i, s := range ss {
		xs[i] = ParseStringExpr(depth, s)
	}
	return QuotedTextExpr(depth, xs...)
}

func Text(s string) *ast.Text {
	return &ast.Text{Kind: ast.TextContent, Value: s}
}

func WSpace() *ast.Text {
	return &ast.Text{Kind: ast.TextSpace}
}

func NBSP() *ast.Text {
	return &ast.Text{Kind: ast.TextNBSP}
}

func Math(x string) *ast.Text {
	return &ast.Text{Kind: ast.TextMath, Value: x}
}

func Comma() *ast.Text {
	return &ast.Text{Kind: ast.TextComma}
}

func UnparsedText(s string) ast.Expr {
	return &ast.UnparsedText{
		Kind:  token.String,
		Value: s,
	}
}

func Ident(s string) ast.Expr {
	return &ast.Ident{
		Name: s,
		Obj:  nil,
	}
}

func Concat(x, y ast.Expr) ast.Expr {
	return &ast.ConcatExpr{X: x, Y: y}
}

func ExprString(x ast.Expr) string {
	switch v := x.(type) {
	case *ast.Ident:
		return "Ident(" + v.Name + ")"
	case *ast.Number:
		return "Number(" + v.Value + ")"
	case *ast.UnparsedText:
		if v.Kind == token.String {
			return "UnparsedText(\"" + v.Value + "\")"
		} else {
			return "UnparsedText({" + v.Value + "})"
		}
	case *ast.Text:
		switch v.Kind {
		case ast.TextSpace:
			return "<space>"
		case ast.TextNBSP:
			return "<NBSP>"
		case ast.TextHyphen:
			return "<hyphen>"
		case ast.TextComma:
			return "<comma>"
		case ast.TextMath:
			return "$" + v.Value + "$"
		case ast.TextContent:
			return fmt.Sprintf("%q", v.Value)
		default:
			return "Text[" + v.Kind.String() + "](" + v.Value + ")"
		}

	case *ast.ParsedText:
		sb := strings.Builder{}
		delim := "quote"
		if v.Delim == ast.BraceDelimiter {
			delim = "brace"
		}
		sb.WriteString(fmt.Sprintf("ParsedText[%d, %s](", v.Depth, delim))
		for i, val := range v.Values {
			sb.WriteString(ExprString(val))
			if i < len(v.Values)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString(")")
		return sb.String()

	case *ast.ConcatExpr:
		return ExprString(v.X) + " # " + ExprString(v.Y)

	default:
		return fmt.Sprintf("UnknownExpr(%v)", v)
	}
}

func WithBibKeys(ts ...string) func(decl *ast.BibDecl) {
	return func(b *ast.BibDecl) {
		if len(ts) > 0 {
			b.Key = &ast.Ident{Name: ts[0]}
			ts = ts[1:]
		}
		for _, k := range ts {
			b.ExtraKeys = append(b.ExtraKeys, &ast.Ident{Name: k})
		}
	}
}

func WithBibTags(key string, val ast.Expr, rest ...interface{}) func(decl *ast.BibDecl) {
	if len(rest)%2 != 0 {
		panic("WithBibTags must have even number of strings for key-val pairs")
	}
	for i := 0; i < len(rest); i += 2 {
		k := rest[i]
		v := rest[i+1]
		if _, ok := k.(string); !ok {
			panic("need string at index: " + strconv.Itoa(i))
		}
		if _, ok := v.(ast.Expr); !ok {
			panic(fmt.Sprintf("need ast.Expr at index: %d of WithBibTags, got: %v", i+1, v))
		}
	}
	return func(b *ast.BibDecl) {
		b.Tags = append(b.Tags, &ast.TagStmt{
			Name:    key,
			RawName: key,
			Value:   val,
		})
		for i := 0; i < len(rest); i += 2 {
			k, v := rest[i].(string), rest[i+1].(ast.Expr)
			tag := &ast.TagStmt{
				Name:    k,
				RawName: k,
				Value:   v,
			}
			b.Tags = append(b.Tags, tag)
		}
	}
}
