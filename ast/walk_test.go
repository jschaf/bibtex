package ast

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type walkOverrideFunc = func(Node) (bool, WalkStatus, error)

func TestWalk(t *testing.T) {
	collectTypesWalker := func(root Node, overrideFunc walkOverrideFunc) (string, error) {
		sb := &strings.Builder{}
		sb.Grow(128)
		err := Walk(root, func(n Node, isEntering bool) (WalkStatus, error) {
			if ok, walkStatus, err := overrideFunc(n); ok {
				return walkStatus, err
			}
			if isEntering {
				_, _ = fmt.Fprintf(sb, "<%T>", n)
				if t, ok := n.(*Text); ok {
					_, _ = fmt.Fprintf(sb, "%s", t.Value)
				}
			} else {
				_, _ = fmt.Fprintf(sb, "</%T>", n)
			}
			return WalkContinue, nil
		})
		return sb.String(), err
	}

	tests := []struct {
		name     string
		node     Node
		override walkOverrideFunc
		want     string
	}{
		{
			name: "visits all in depth first order",
			node: &ParsedText{
				Depth: 0,
				Delim: BraceDelimiter,
				Values: []Expr{
					&Text{Value: "first"},
					&ParsedText{
						Values: []Expr{&Text{Value: "second"}},
					},
				},
			},
			override: func(_ Node) (bool, WalkStatus, error) { return false, WalkContinue, nil },
			want: strings.Join(
				[]string{
					"<*ast.ParsedText>",
					"<*ast.Text>first</*ast.Text>",
					"<*ast.ParsedText><*ast.Text>second</*ast.Text></*ast.ParsedText>",
					"</*ast.ParsedText>",
				},
				""),
		},
		{
			name: "visits all in depth first order",
			node: &ParsedText{
				Depth: 0,
				Delim: BraceDelimiter,
				Values: []Expr{
					&Text{Value: "first"},
					&ParsedText{
						Values: []Expr{&Text{Value: "second"}},
					},
				},
			},
			override: func(n Node) (bool, WalkStatus, error) {
				if t, ok := n.(*Text); ok && t.Value == "second" {
					return true, WalkStop, nil
				}
				return false, WalkContinue, nil
			},
			want: strings.Join(
				[]string{
					"<*ast.ParsedText>",
					"<*ast.Text>first</*ast.Text>",
					"<*ast.ParsedText>",
				},
				""),
		},
		{
			name: "visits authors",
			node: Authors{
				&Author{
					First: &Text{Value: "first"},
					Last: &ParsedText{
						Values: []Expr{&Text{Value: "last-first"}, &Text{Value: "last-second"}},
					},
				},
			},
			override: func(_ Node) (bool, WalkStatus, error) { return false, WalkContinue, nil },
			want: strings.Join(
				[]string{
					"<ast.Authors>",
					"<*ast.Author><*ast.Text>first</*ast.Text><*ast.ParsedText><*ast.Text>last-first</*ast.Text><*ast.Text>last-second</*ast.Text></*ast.ParsedText></*ast.Author>",
					"</ast.Authors>",
				},
				""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collectTypesWalker(tt.node, tt.override)
			if err != nil {
				t.Errorf("Walk() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Walk() mismatch (-want +got)\n%s", diff)
			}
		})
	}
}
