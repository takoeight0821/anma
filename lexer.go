package main

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

func Lex(source string) ([]Token, error) {
	l := lexer{
		source:  []rune(source),
		tokens:  []Token{},
		start:   0,
		current: 0,
		line:    1,
	}

	var err error

	for !l.isAtEnd() {
		err = errors.Join(err, l.scanToken())
	}

	l.tokens = append(l.tokens, Token{EOF, "", l.line, nil})
	return l.tokens, err
}

type lexer struct {
	source []rune
	tokens []Token

	start   int // start of current lexeme
	current int // current position in source
	line    int // current line number
}

func (l lexer) isAtEnd() bool {
	return l.current >= len(l.source)
}

func (l lexer) peek() rune {
	if l.isAtEnd() {
		return '\x00'
	}
	return l.source[l.current]
}

func (l *lexer) advance() rune {
	l.current++
	return l.source[l.current-1]
}

func (l *lexer) addToken(kind TokenKind, literal any) {
	text := string(l.source[l.start:l.current])
	l.tokens = append(l.tokens, Token{kind, text, l.line, literal})
}

func (l *lexer) scanToken() error {
	l.start = l.current
	c := l.advance()
	switch c {
	case ' ', '\r', '\t':
		// ignore whitespace
		return nil
	case '\n':
		l.line++
		return nil
	case '"':
		return l.string()
	default:
		if k, ok := reservedSymbols[c]; ok {
			l.addToken(k, nil)
			return nil
		}
		if isDigit(c) {
			return l.integer()
		}
		if isAlpha(c) {
			return l.identifier()
		}
		if isSymbol(c) {
			return l.operator()
		}
	}
	return fmt.Errorf("unexpected character: %c", c)
}

func (l *lexer) string() error {
	for l.peek() != '"' && !l.isAtEnd() {
		if l.peek() == '\n' {
			l.line++
		}
		if l.peek() == '\\' {
			l.advance()
		}
		l.advance()
	}

	if l.isAtEnd() {
		return fmt.Errorf("unterminated string")
	}

	r := l.advance()

	if r != '"' {
		return fmt.Errorf("unterminated string")
	}

	value := string(l.source[l.start+1 : l.current-1])
	l.addToken(STRING, value)
	return nil
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func (l *lexer) integer() error {
	for isDigit(l.peek()) {
		l.advance()
	}

	value, err := strconv.Atoi(string(l.source[l.start:l.current]))
	l.addToken(INTEGER, value)
	return err
}

func isAlpha(c rune) bool {
	return unicode.IsLetter(c) || c == '_'
}

func (l *lexer) identifier() error {
	for isAlpha(l.peek()) || isDigit(l.peek()) {
		l.advance()
	}

	value := string(l.source[l.start:l.current])

	if k, ok := keywords[value]; ok {
		l.addToken(k, nil)
	} else {
		l.addToken(IDENT, value)
	}
	return nil
}

var keywords = map[string]TokenKind{
	"type": TYPE,
	"def":  DEF,
	"case": CASE,
}

func isSymbol(c rune) bool {
	_, isReserved := reservedSymbols[c]
	return c != '_' && !isReserved && (unicode.IsSymbol(c) || unicode.IsPunct(c))
}

var reservedSymbols = map[rune]TokenKind{
	'(': LEFT_PAREN,
	')': RIGHT_PAREN,
	'{': LEFT_BRACE,
	'}': RIGHT_BRACE,
	'[': LEFT_BRACKET,
	']': RIGHT_BRACKET,
	':': COLON,
	',': COMMA,
	'.': DOT,
	';': SEMICOLON,
	'#': SHARP,
}

func (l *lexer) operator() error {
	for isSymbol(l.peek()) {
		l.advance()
	}

	value := string(l.source[l.start:l.current])
	if k, ok := keywords[value]; ok {
		l.addToken(k, nil)
	} else {
		l.addToken(OPERATOR, value)
	}
	return nil
}

//go:generate go run golang.org/x/tools/cmd/stringer@v0.13.0 -type=TokenKind
type TokenKind int

const (
	EOF TokenKind = iota

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
	TYPE
	DEF
	CASE
)

type Token struct {
	Kind    TokenKind
	Lexeme  string
	Line    int
	Literal any
}

func (t Token) String() string {
	if t.Literal == nil {
		return fmt.Sprintf("[%v %v %v]", t.Kind, t.Lexeme, t.Line)
	} else {
		return fmt.Sprintf("[%v %v %v %#v]", t.Kind, t.Lexeme, t.Line, t.Literal)
	}
}
