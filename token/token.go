package token

//go:generate go run golang.org/x/tools/cmd/stringer@v0.13.0 -type=Kind
type Kind int

const (
	EOF Kind = iota

	// Single-character tokens.
	LEFT_PAREN
	RIGHT_PAREN
	LEFT_BRACE
	RIGHT_BRACE
	LEFT_BRACKET
	RIGHT_BRACKET
	COLON
	COMMA
	DOT
	SEMICOLON
	SHARP

	// Literals and identifiers.
	IDENT
	OPERATOR
	INTEGER
	STRING

	// Keywords.
	ARROW
	CASE
	DEF
	EQUAL
	FN
	LET
	TYPE
)

type Token struct {
	Kind    Kind
	Lexeme  string
	Line    int
	Literal any
}

func (t Token) String() string {
	return t.Lexeme
}
