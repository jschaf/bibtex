// Package token defines constants representing the lexical tokens of the bibtex
// language and basic operations on tokens (printing, predicates).
package token

import "strconv"

// References
// - http://www.bibtex.org/Format/
// - http://mirror.utexas.edu/ctan/biblio/bibtex/base/btxdoc.pdf
// - http://maverick.inria.fr/~Xavier.Decoret/resources/xdkbibtex/bibtex_summary.html
// - http://ctan.math.illinois.edu/info/bibtex/tamethebeast/ttb_en.pdf

// Token is the set of lexical tokens for bibtex.
type Token int

const (
	Illegal Token = iota
	EOF
	TexComment // % foo

	commandBegin
	Abbrev   // @STRING, @string
	Comment  // @COMMENT, @comment
	Preamble // @PREAMBLE, @pReAmble
	BibEntry // @article, @book, etc
	commandEnd

	// Identifiers and basic type literals that represent a class of literals.
	literalBegin
	Ident       // author
	String      // "abc"
	BraceString // {abc}
	Number      // 2005
	literalEnd

	// Tokens delimiting strings or contained in a bibtex string literal.
	stringLiteralBegin
	DoubleQuote    // " - delimits a string
	StringLBrace   // {
	StringRBrace   // }
	StringSpace    // any consecutive whitespace '\n', '\r', '\t', ' ' in a bibtex string
	StringNBSP     // ~ - non-breakable space in LaTeX
	StringContents // anything inside a string
	Hyphen         // - a hyphen counts as a token separator when parsing names
	SpecialToken   // {\ <anything } - any LBrace followed by a backslash through the RBrace
	StringMath     // $...$
	StringComma    // , - useful for parsing author names
	stringLiteralEnd

	// Operators and delimiters.
	operatorBegin
	Assign // =
	LParen // (
	LBrace // {
	RParen // )
	RBrace // }
	Concat // #
	Comma  // ,
	operatorEnd
)

var tokens = [...]string{
	Illegal:    "Illegal",
	EOF:        "EOF",
	TexComment: "TexComment",

	// Commands
	Abbrev:   "Abbrev",
	Comment:  "Comment",
	Preamble: "Preamble",
	BibEntry: "BibEntry",

	// Literal
	Ident:       "Ident",
	String:      "String",
	BraceString: "BraceString",
	Number:      "Number",

	// String literals
	DoubleQuote:    "DoubleQuote",
	StringLBrace:   "StringLBrace",
	StringRBrace:   "StringRBrace",
	StringSpace:    "StringSpace",
	StringNBSP:     "NBSP",
	StringContents: "StringContents",
	Hyphen:         "Hyphen",
	SpecialToken:   "SpecialToken",
	StringMath:     "StringMath",
	StringComma:    "StringComma",

	// Operators and delimiters
	Assign: "Assign",
	LParen: "LParen",
	LBrace: "LBrace",
	RParen: "RParen",
	RBrace: "RBrace",
	Concat: "Concat",
	Comma:  "Comma",
}

func (tok Token) String() string {
	s := ""
	if 0 <= tok && tok < Token(len(tokens)) {
		s = tokens[tok]
	}
	if s == "" {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}
	return s
}

// IsLiteral returns true for tokens corresponding to identifiers and basic type
// literals. It returns false otherwise.
func (tok Token) IsLiteral() bool {
	return literalBegin < tok && tok < literalEnd
}

// IsStringLiteral returns true for tokens corresponding to tokens within a
// string literal.  It returns false otherwise.
func (tok Token) IsStringLiteral() bool {
	return stringLiteralBegin < tok && tok < stringLiteralEnd
}

// IsOperator returns true for tokens corresponding to operators and delimiters.
// It returns false otherwise.
func (tok Token) IsOperator() bool {
	return operatorBegin < tok && tok < operatorEnd
}

// IsCommand returns true for tokens corresponding to commands. It returns false
// otherwise.
func (tok Token) IsCommand() bool {
	return commandBegin < tok && tok < commandEnd
}
