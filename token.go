package main

//go:generate go run golang.org/x/tools/cmd/stringer@v0.13.0 -type=TokenKind
type TokenKind int

const (
	EOF TokenKind = iota

	// Single-character tokens.
	LEFTPAREN
	RIGHTPAREN
	LEFTBRACE
	RIGHTBRACE
	LEFTBRACKET
	RIGHTBRACKET
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
	INFIX
	INFIXL
	INFIXR
	LET
	TYPE
)

type Token struct {
	Kind    TokenKind
	Lexeme  string
	Line    int
	Literal any
}

func (t Token) String() string {
	return t.Lexeme
}

func (t Token) Base() Token {
	return t
}

var _ Node = Token{}
