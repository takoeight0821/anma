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
	BAR
	CASE
	DEF
	EQUAL
	FN
	INFIX
	INFIXL
	INFIXR
	LET
	TYPE
	PRIM
)

type Token struct {
	Kind    TokenKind
	Lexeme  string
	Line    int
	Literal any
}

func (t Token) Pretty() string {
	if (t.Kind == IDENT || t.Kind == OPERATOR) && t.Literal != nil {
		return fmt.Sprintf("%s.%#v", t.Lexeme, t.Literal)
	}
	return t.Lexeme
}

func (t Token) String() string {
	return fmt.Sprintf("{%v, %q, %d, %v}", t.Kind, t.Lexeme, t.Line, t.Literal)
}

func (t Token) Base() Token {
	return t
}
