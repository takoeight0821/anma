package lexer

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"

	"github.com/takoeight0821/anma/token"
)

func Lex(source string) ([]token.Token, error) {
	l := lexer{
		source:  []rune(source),
		tokens:  []token.Token{},
		start:   0,
		current: 0,
		line:    1,
	}

	var err error

	for !l.isAtEnd() {
		err = errors.Join(err, l.scanToken())
	}

	l.tokens = append(l.tokens, token.Token{Kind: token.EOF, Lexeme: "", Line: l.line, Literal: nil})
	return l.tokens, err
}

type lexer struct {
	source []rune
	tokens []token.Token

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

func (l *lexer) addToken(kind token.TokenKind, literal any) {
	text := string(l.source[l.start:l.current])
	l.tokens = append(l.tokens, token.Token{Kind: kind, Lexeme: text, Line: l.line, Literal: literal})
}

type UnexpectedCharacterError struct {
	Line int
	Char rune
}

func (e UnexpectedCharacterError) Error() string {
	return fmt.Sprintf("unexpected character: %c at line %d", e.Char, e.Line)
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
	return UnexpectedCharacterError{Line: l.line, Char: c}
}

type UnterminatedStringError struct {
	Line int
}

func (e UnterminatedStringError) Error() string {
	return fmt.Sprintf("unterminated string at line %d", e.Line)
}

func (l *lexer) string() error {
	for l.peek() != '"' && !l.isAtEnd() {
		if l.peek() == '\n' {
			l.line++
		}
		if l.peek() == '\\' {
			l.advance()
			if l.isAtEnd() {
				return UnterminatedStringError{Line: l.line}
			}
		}
		l.advance()
	}

	if l.isAtEnd() {
		return UnterminatedStringError{Line: l.line}
	}

	r := l.advance()

	if r != '"' {
		return UnterminatedStringError{Line: l.line}
	}

	value := string(l.source[l.start+1 : l.current-1])
	l.addToken(token.STRING, value)
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
	if err != nil {
		return fmt.Errorf("invalid integer: %w", err)
	}
	l.addToken(token.INTEGER, value)
	return nil
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
		l.addToken(token.IDENT, nil)
	}
	return nil
}

var keywords = map[string]token.TokenKind{
	"->":     token.ARROW,
	"=":      token.EQUAL,
	"case":   token.CASE,
	"def":    token.DEF,
	"fn":     token.FN,
	"infix":  token.INFIX,
	"infixl": token.INFIXL,
	"infixr": token.INFIXR,
	"let":    token.LET,
	"type":   token.TYPE,
	"prim":   token.PRIM,
}

func isSymbol(c rune) bool {
	_, isReserved := reservedSymbols[c]
	return c != '_' && !isReserved && (unicode.IsSymbol(c) || unicode.IsPunct(c))
}

// These characters are reserved symbols, but they are not included in operator.
var reservedSymbols = map[rune]token.TokenKind{
	'(': token.LEFTPAREN,
	')': token.RIGHTPAREN,
	'{': token.LEFTBRACE,
	'}': token.RIGHTBRACE,
	'[': token.LEFTBRACKET,
	']': token.RIGHTBRACKET,
	':': token.COLON,
	',': token.COMMA,
	'.': token.DOT,
	';': token.SEMICOLON,
	'#': token.SHARP,
}

func (l *lexer) operator() error {
	for isSymbol(l.peek()) {
		l.advance()
	}

	value := string(l.source[l.start:l.current])
	if k, ok := keywords[value]; ok {
		l.addToken(k, nil)
	} else {
		l.addToken(token.OPERATOR, nil)
	}
	return nil
}
