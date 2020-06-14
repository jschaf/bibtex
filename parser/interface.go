// This file contains the exported entry points for invoking the parser.
package parser

import (
	"bytes"
	"errors"
	gotok "go/token"
	"io"
	"io/ioutil"

	"github.com/jschaf/b2/pkg/bibtex/ast"
)

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
//
func readSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch s := src.(type) {
		case string:
			return []byte(s), nil
		case []byte:
			return s, nil
		case *bytes.Buffer:
			// is io.Reader, but src is already available in []byte form
			if s != nil {
				return s.Bytes(), nil
			}
		case io.Reader:
			return ioutil.ReadAll(s)
		}
		return nil, errors.New("invalid source")
	}
	return ioutil.ReadFile(filename)
}

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional parser
// functionality.
type Mode uint

const (
	ParseComments     Mode = 1 << iota // parse comments and add them to AST
	ParseStrings                       // parse contents of strings, "{F}oo{"}"
	Trace                              // print a trace of parsed productions
	DeclarationErrors                  // report declaration errors
	AllErrors                          // report all errors (not just the first 10 on different lines)
)

// ParseFile parses the source code of a single Go source file and returns
// the corresponding ast.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
//
// If src != nil, ParseFile parses the source from src and the filename is
// only used when recording position information. The type of the argument
// for the src parameter must be string, []byte, or io.Reader.
// If src == nil, ParseFile parses the file specified by filename.
//
// The mode parameter controls the amount of source text parsed and other
// optional parser functionality. Position information is recorded in the
// file set fset, which must not be nil.
//
// If the source couldn't be read, the returned AST is nil and the error
// indicates the specific failure. If the source was read but syntax
// errors were found, the result is a partial AST (with ast.Bad* nodes
// representing the fragments of erroneous source code). Multiple errors
// are returned via a scanner.ErrorList which is sorted by source position.
func ParseFile(fset *gotok.FileSet, filename string, src interface{}, mode Mode) (f *ast.File, err error) {
	if fset == nil {
		panic("parser.ParseFile: no token.FileSet provided (fset == nil)")
	}

	// get source
	text, err := readSource(filename, src)
	if err != nil {
		return nil, err
	}

	var p parser
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		// set result values
		if f == nil {
			// source is not a valid Go source file - satisfy
			// ParseFile API and return a valid (but) empty
			// *ast.File
			f = &ast.File{
				Name:  filename,
				Scope: ast.NewScope(nil),
			}
		}

		p.errors.Sort()
		err = p.errors.Err()
	}()

	// parse source
	p.init(fset, filename, text, mode)
	f = p.parseFile()

	return
}

func ParseExpr(str string) (ast.Expr, error) {
	fset := gotok.NewFileSet()
	// use '=' to trick parser into treating '{' as a string
	src := []byte("=" + str)
	var p parser
	p.init(fset, "", src, ParseStrings)
	p.next() // consume the '='
	expr := p.parseExpr()
	p.errors.Sort()
	err := p.errors.Err()
	if err != nil {
		return nil, err
	}
	return expr, nil
}

// ParsePackage calls ParseFile for all files specified by paths.
//
// The mode bits are passed to ParseFile unchanged.
//
// If a parse error occurred, an incomplete package and the first error
// encountered are returned.
func ParsePackage(paths []string, mode Mode) (pkg *ast.Package, first error) {
	fset := gotok.NewFileSet()
	pkg = &ast.Package{}
	for _, filename := range paths {
		if src, err := ParseFile(fset, filename, nil, mode); err == nil {
			pkg.Files[filename] = src
		} else if first == nil {
			first = err
		}
	}
	return
}
