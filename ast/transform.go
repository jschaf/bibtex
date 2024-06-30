package ast

import "fmt"

type Transformer interface {
	Transform(Node) error
}

type SimplifyTagTransformer struct{}

func (s SimplifyTagTransformer) Transform(node Node) error {
	err := Walk(node, func(n Node, isEntering bool) (WalkStatus, error) {
		if _, ok := n.(*TagStmt); !ok || !isEntering {
			return WalkSkipChildren, nil
		}
		tag := n.(*TagStmt)
		if txt, ok := tag.Value.(*ParsedText); ok {
			tag.Value = SimplifyParsedText(txt)
		}
		return WalkSkipChildren, nil
	})
	if err != nil {
		return fmt.Errorf("simplify tag transform: %w", err)
	}
	return nil
}

func SimplifyParsedText(_ *ParsedText) *Text {
	return nil
}
