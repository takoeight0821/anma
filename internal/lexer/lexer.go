package lexer

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"

	"github.com/takoeight0821/anma/internal/token"
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
			if l.isAtEnd() {
				return fmt.Errorf("unterminated string")
			}
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
	l.addToken(token.INTEGER, value)
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
