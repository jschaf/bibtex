// This file implements scopes and the objects they contain.

package ast

import (
	"bytes"
	"fmt"
	"go/token"
)

// A Scope maintains the set of named language entities declared in the scope
// and a link to the immediately surrounding (outer) scope.
type Scope struct {
	Outer   *Scope
	Objects map[string]*Object
}

// NewScope creates a new scope nested in the outer scope.
func NewScope(outer *Scope) *Scope {
	const n = 4 // initial scope capacity
	return &Scope{outer, make(map[string]*Object, n)}
}

// Lookup returns the object with the given name if it is found in scope s,
// otherwise it returns nil. Outer scopes are ignored.
func (s *Scope) Lookup(name string) *Object {
	return s.Objects[name]
}

// Insert attempts to insert a named object obj into the scope s. If the scope
// already contains an object alt with the same name, Insert leaves the scope
// unchanged and returns alt. Otherwise it inserts obj and returns nil.
func (s *Scope) Insert(obj *Object) (alt *Object) {
	if alt = s.Objects[obj.Name]; alt == nil {
		s.Objects[obj.Name] = obj
	}
	return
}

// Debugging support
func (s *Scope) String() string {
	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "scope %p {", s)
	if s != nil && len(s.Objects) > 0 {
		_, _ = fmt.Fprintln(&buf)
		for _, obj := range s.Objects {
			_, _ = fmt.Fprintf(&buf, "\t%s %s\n", obj.Kind, obj.Name)
		}
	}
	_, _ = fmt.Fprintf(&buf, "}\n")
	return buf.String()
}

// ----------------------------------------------------------------------------
// Objects

// An Object describes a named language entity such as a bibtex entry, or string
// abbreviation.
//
// The Data fields contains object-specific data:
//
//	Kind    Data type         Data value
//	Pkg     *Scope            package scope
//	Con     int               iota for the respective declaration
type Object struct {
	Kind ObjKind
	Name string      // declared name
	Decl interface{} // corresponding BibDecl, AbbrevDecl, or nil
}

// NewObj creates a new object of a given kind and name.
func NewObj(kind ObjKind, name string) *Object {
	return &Object{Kind: kind, Name: name}
}

// Pos computes the source position of the declaration of an object name.
// The result may be an invalid position if it cannot be computed
// (obj.Decl may be nil or not correct).
func (obj *Object) Pos() token.Pos {
	name := obj.Name
	switch d := obj.Decl.(type) {
	case *BibDecl:
		if d.Key != nil && d.Key.Name == name {
			return d.Key.Pos()
		}
	case *Scope:
		// predeclared object - nothing to do for now
	}
	return token.NoPos
}

// ObjKind describes what an object represents.
type ObjKind int

// The list of possible Object kinds.
const (
	Bad    ObjKind = iota // for error handling
	Entry                 // bibtex entry, usually from a crossref
	Abbrev                // abbreviation
)

var objKindStrings = [...]string{
	Bad:    "bad",
	Entry:  "entry",
	Abbrev: "abbrev",
}

func (kind ObjKind) String() string { return objKindStrings[kind] }
