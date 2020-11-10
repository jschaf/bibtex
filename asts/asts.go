// Packages asts contains utilities for constructing and manipulating ASTs.
package asts

import (
	"fmt"
	"github.com/jschaf/bibtex/ast"
	"github.com/jschaf/bibtex/scanner"
	"github.com/jschaf/bibtex/token"
	gotok "go/token"
	"strconv"
	"strings"
)

func UnparsedBraceText(s string) *ast.UnparsedText {
	return &ast.UnparsedText{
		Type:  token.BraceString,
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
// - If the string begins with '\' and has an alphabetical char, convert to
//   command ast.TextMacro.
// - Otherwise, convert to ast.Text.
func BraceText(depth int, ss ...interface{}) *ast.ParsedText {
	xs := make([]ast.Expr, len(ss))
	for i, s := range ss {
		xs[i] = ParseAny(s)
	}
	return &ast.ParsedText{
		Depth:  depth,
		Delim:  ast.BraceDelimiter,
		Values: xs,
	}
}

// ParseAny converts s into an ast.Expr or panics.
func ParseAny(s interface{}) ast.Expr {
	switch x := s.(type) {
	case ast.Expr:
		return x
	case string:
		return ParseStringExpr(0, x)
	default:
		panic(fmt.Sprintf("unsupported type for ParseAny: %v", x))
	}
}

func ParseStringExpr(depth int, s string) ast.Expr {
	switch {
	case strings.TrimSpace(s) == "":
		return Space()
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
				xs[idx] = Space()
				idx++
			}
		}
		return BraceTextExpr(depth+1, xs...)
	case s == ",":
		return Comma()
	case len(s) >= 2 && s[0] == '\\' && scanner.IsAsciiLetter(rune(s[1])):
		return Macro(s[1:]) // drop backslash
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
	return &ast.Text{Value: s}
}

func Space() *ast.TextSpace {
	return &ast.TextSpace{Value: " "}
}

func NBSP() *ast.TextNBSP {
	return &ast.TextNBSP{}
}

func Math(x string) *ast.TextMath {
	return &ast.TextMath{Value: x}
}

func Comma() *ast.TextComma {
	return &ast.TextComma{}
}

func Macro(name string, params ...interface{}) *ast.TextMacro {
	vals := make([]ast.Expr, len(params))
	for i, param := range params {
		vals[i] = ParseAny(param)
	}
	return &ast.TextMacro{Name: name, Values: vals}
}

func Escaped(c rune) *ast.TextEscaped {
	return &ast.TextEscaped{Value: string(c)}
}

func UnparsedText(s string) ast.Expr {
	return &ast.UnparsedText{
		Type:  token.String,
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
		if v.Type == token.String {
			return "UnparsedText(\"" + v.Value + "\")"
		} else {
			return "UnparsedText({" + v.Value + "})"
		}
	case *ast.TextSpace:
		return "<space>"
	case *ast.TextNBSP:
		return "<NBSP>"
	case *ast.TextHyphen:
		return "<hyphen>"
	case *ast.TextComma:
		return "<comma>"
	case *ast.TextMath:
		return "$" + v.Value + "$"
	case *ast.TextEscaped:
		return `\` + v.Value
	case *ast.Text:
		return fmt.Sprintf("%q", v.Value)

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

	case *ast.TextMacro:
		sb := strings.Builder{}
		sb.WriteString("TextMacro(")
		sb.WriteByte('\\')
		sb.WriteString(v.Name)
		if len(v.Values) == 0 {
			if v.RBrace != gotok.NoPos {
				sb.WriteString("{}")
			}
			return sb.String()
		}
		sb.WriteString("{")
		for i, value := range v.Values {
			sb.WriteString(ExprString(value))
			if i < len(v.Values)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("})")
		return sb.String()

	default:
		return fmt.Sprintf("UnknownExpr(%v)", v)
	}
}

func WithBibType(s string) func(decl *ast.BibDecl) {
	return func(b *ast.BibDecl) {
		b.Type = s
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
