package token

// Accent is the marker rune of a LaTeX accent.
type Accent rune

const (
	AccentAcute      Accent = '\''
	AccentCedilla    Accent = 'c'
	AccentCircumflex Accent = '^'
	AccentDot        Accent = '.'
	AccentGrave      Accent = '`'
	AccentTilde      Accent = '~'
	AccentUmlaut     Accent = '"'
)
