package ast

type WalkStatus int

const (
	WalkStop         WalkStatus = iota // stop walking immediately
	WalkContinue                       // continue walking
	WalkSkipChildren                   // continue walking
)

// Walker is a function that's called on every node recursively as part of the
// traversal for Walk.
type Walker = func(n Node, isEntering bool) (WalkStatus, error)

// Walk walks the AST using depth-first-search.
//
// Specifically, walker is first called on a node with isEntering set to true.
// Then each child is visited recursively. Finally, walker is called with
// isEntering set to false.
//
// The traversal stops whenever the walker returns WalkStop or an error.
func Walk(n Node, w Walker) error {
	_, err := walkHelper(n, w)
	return err
}

func walkHelper(n Node, walker Walker) (WalkStatus, error) {
	// Call walker with isEntering == true.
	st1, err1 := walker(n, true)
	if st1 == WalkStop || err1 != nil {
		return st1, err1
	}

	// Recursive case only applies if we aren't skipping children:
	if st1 != WalkSkipChildren {
		switch t := n.(type) {
		case *File:
			for _, entry := range t.Entries {
				if st, err := walkHelper(entry, walker); st == WalkStop || err != nil {
					return st, err
				}
			}
		case *Package:
			for _, file := range t.Files {
				if st, err := walkHelper(file, walker); st == WalkStop || err != nil {
					return st, err
				}
			}
		case *BibDecl:
			for _, tag := range t.Tags {
				if st, err := walkHelper(tag, walker); st == WalkStop || err != nil {
					return st, err
				}
			}
		case *PreambleDecl:
			if st, err := walkHelper(t.Text, walker); st == WalkStop || err != nil {
				return st, err
			}
		case *ParsedText:
			for _, child := range t.Values {
				if st, err := walkHelper(child, walker); st == WalkStop || err != nil {
					return st, err
				}
			}
		case *ConcatExpr:
			if st, err := walkHelper(t.X, walker); st == WalkStop || err != nil {
				return st, err
			}
			if st, err := walkHelper(t.Y, walker); st == WalkStop || err != nil {
				return st, err
			}
		case *MacroText:
			for _, child := range t.Values {
				if st, err := walkHelper(child, walker); st == WalkStop || err != nil {
					return st, err
				}
			}
		}
	}

	// Call walker with isEntering == false.
	if st, err := walker(n, false); st == WalkStop || err != nil {
		return st, err
	}
	return WalkContinue, nil
}
