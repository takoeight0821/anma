package token

import "fmt"

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
	if t.Kind == IDENT && t.Literal != nil {
		return fmt.Sprintf("%s.%#v", t.Lexeme, t.Literal)
	}
	return t.Lexeme
}

func (t Token) Base() Token {
	return t
}
