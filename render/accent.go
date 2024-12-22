package render

import (
	"fmt"
	"strings"
)

// AccentType represents the different LaTeX accents
type AccentType string

const (
	Grave      AccentType = "`"
	Acute      AccentType = "'"
	Circumflex AccentType = "^"
	Umlaut     AccentType = `"`
	Tilde      AccentType = "~"
	Cedilla    AccentType = "c"
	Dot        AccentType = "."
)

var allAccents = map[string]struct{}{
	string(Grave):      {},
	string(Acute):      {},
	string(Circumflex): {},
	string(Umlaut):     {},
	string(Tilde):      {},
	string(Cedilla):    {},
	string(Dot):        {},
}

// NewAccentType converts a string representation to an AccentType
// It returns the AccentType and an error if the input is invalid
func NewAccentType(accentStr string) (AccentType, error) {
	// Normalize the input string to lowercase
	accentStr = strings.ToLower(strings.TrimSpace(accentStr))

	// Look up the accent type
	if _, isValid := allAccents[accentStr]; !isValid {
		// Generate a helpful error message with available options
		availableAccents := make([]string, 0, len(allAccents))
		for k := range allAccents {
			availableAccents = append(availableAccents, k)
		}

		return "", fmt.Errorf("invalid accent type. Available accent types are: %v",
			strings.Join(availableAccents, ", "))
	}

	return AccentType(accentStr), nil

}

// Mapping of base characters to their accented versions
var accentMap = map[string]rune{
	// Grave accent (`)
	"`a": 'à', "`e": 'è', "`i": 'ì', "`o": 'ò', "`u": 'ù',
	"`A": 'À', "`E": 'È', "`I": 'Ì', "`O": 'Ò', "`U": 'Ù',

	// Acute accent (')
	"'a": 'á', "'e": 'é', "'i": 'í', "'o": 'ó', "'u": 'ú', "'y": 'ý',
	"'A": 'Á', "'E": 'É', "'I": 'Í', "'O": 'Ó', "'U": 'Ú', "'Y": 'Ý',

	// Circumflex accent (^)
	"^a": 'â', "^e": 'ê', "^i": 'î', "^o": 'ô', "^u": 'û',
	"^A": 'Â', "^E": 'Ê', "^I": 'Î', "^O": 'Ô', "^U": 'Û',

	// Umlaut/diaeresis (")
	"\"a": 'ä', "\"e": 'ë', "\"i": 'ï', "\"o": 'ö', "\"u": 'ü',
	"\"A": 'Ä', "\"E": 'Ë', "\"I": 'Ï', "\"O": 'Ö', "\"U": 'Ü',

	// Tilde (~)
	"~a": 'ã', "~n": 'ñ', "~o": 'õ',
	"~A": 'Ã', "~N": 'Ñ', "~O": 'Õ',

	// Cedilla (c)
	"cc": 'ç', "cC": 'Ç',

	// Dot (.)
	".c": 'ċ', ".e": 'ė', ".g": 'ġ', ".i": 'ı', ".z": 'ż',
	".C": 'Ċ', ".E": 'Ė', ".G": 'Ġ', ".I": 'İ', ".Z": 'Ż',
}

// Check if the accent exists for the given base character

// ConvertLatexAccent converts a base character with a LaTeX accent to its UTF-8 equivalent
func FmtAccent(baseChar rune, accent AccentType) (rune, error) {
	mapKey := string(accent) + string(baseChar)

	accentedChar, exists := accentMap[mapKey]

	if !exists {
		// Generate a helpful error message
		availableChars := []string{}
		for key := range accentMap {
			if strings.HasPrefix(key, string(accent)) {
				availableChars = append(availableChars, key[1:])
			}
		}

		return 0, fmt.Errorf("invalid combination: cannot apply %s accent to character '%c'. "+
			"Available characters for this accent are: %v",
			accent, baseChar, strings.Join(availableChars, ", "))
	}

	return accentedChar, nil
}
