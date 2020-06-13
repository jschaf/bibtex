package namelist

import "strconv"

type NameTok int

const (
	Illegal NameTok = iota
	EOF
	Whitespace  // any whitespace
	String      // Foo
	BraceString // {Foo bar}
	NameSep     // name separator, typically "and"
	Others      // additional unlisted authors, typically "others"
	Comma       // ,
	LBrace      // {
	RBrace      // }
)

var tokens = [...]string{
	Illegal:     "Illegal",
	EOF:         "EOF",
	Whitespace:  "Whitespace",
	String:      "String",
	BraceString: "BraceString",
	NameSep:     "NameSep",
	Others:      "Others",
	Comma:       "Comma",
	LBrace:      "LBrace",
	RBrace:      "RBrace",
}

func (tok NameTok) String() string {
	s := ""
	if 0 <= tok && tok < NameTok(len(tokens)) {
		s = tokens[tok]
	}
	if s == "" {
		s = "nameTok(" + strconv.Itoa(int(tok)) + ")"
	}
	return s
}
