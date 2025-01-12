package render

import (
	"fmt"
	"github.com/jschaf/bibtex/token"
)

// Mapping of base characters to their accented versions
var accentMap = map[string]rune{
	// AccentGrave accent (`)
	"`a": 'à', "`e": 'è', "`i": 'ì', "`o": 'ò', "`u": 'ù',
	"`A": 'À', "`E": 'È', "`I": 'Ì', "`O": 'Ò', "`U": 'Ù',

	// AccentAcute accent (')
	"'a": 'á', "'e": 'é', "'i": 'í', "'o": 'ó', "'u": 'ú', "'y": 'ý',
	"'A": 'Á', "'E": 'É', "'I": 'Í', "'O": 'Ó', "'U": 'Ú', "'Y": 'Ý',

	// AccentCircumflex accent (^)
	"^a": 'â', "^e": 'ê', "^i": 'î', "^o": 'ô', "^u": 'û',
	"^A": 'Â', "^E": 'Ê', "^I": 'Î', "^O": 'Ô', "^U": 'Û',

	// AccentUmlaut/diaeresis (")
	`"a`: 'ä', `"e`: 'ë', `"i`: 'ï', `"o`: 'ö', `"u`: 'ü',
	`"A`: 'Ä', `"E`: 'Ë', `"I`: 'Ï', `"O`: 'Ö', `"U`: 'Ü',

	// AccentTilde (~)
	"~a": 'ã', "~n": 'ñ', "~o": 'õ',
	"~A": 'Ã', "~N": 'Ñ', "~O": 'Õ',

	// AccentCedilla (c)
	"cc": 'ç', "cC": 'Ç',

	// AccentDot (.)
	".c": 'ċ', ".e": 'ė', ".g": 'ġ', ".i": 'ı', ".z": 'ż',
	".C": 'Ċ', ".E": 'Ė', ".G": 'Ġ', ".I": 'İ', ".Z": 'Ż',
}

// RenderAccent renders an accented character.
func RenderAccent(accent token.Accent, text string) (rune, error) {
	if len(text) == 0 {
		return 0, fmt.Errorf("cannot render accent %q for empty text", accent)
	}
	if len(text) > 1 {
		return 0, fmt.Errorf("cannot render accent %q for multi-rune text %q", accent, text)
	}
	if accent == 0 {
		return 0, fmt.Errorf("cannot render accent for empty accent")
	}
	accented, ok := accentMap[(string(accent) + text)]
	if !ok {
		return 0, fmt.Errorf("invalid combination: cannot apply %q accent to character %q", accent, text)
	}
	return accented, nil
}
