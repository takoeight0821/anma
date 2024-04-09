package lexer

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/takoeight0821/anma/token"
)

func Lex(source string) ([]token.Token, error) {
	lexer := lexer{
		source:  source,
		tokens:  []token.Token{},
		start:   0,
		current: 0,
		line:    1,
	}

	var err error

	for !lexer.isAtEnd() {
		err = errors.Join(err, lexer.scanToken())
	}

	lexer.tokens = append(lexer.tokens, token.Token{Kind: token.EOF, Lexeme: "", Line: lexer.line, Literal: nil})

	return lexer.tokens, err
}

type lexer struct {
	source string
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
	runeValue, _ := utf8.DecodeRuneInString(l.source[l.current:])

	return runeValue
}

func (l *lexer) advance() rune {
	runeValue, width := utf8.DecodeRuneInString(l.source[l.current:])
	l.current += width

	return runeValue
}

func (l *lexer) addToken(kind token.Kind, literal any) {
	text := l.source[l.start:l.current]
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
	char := l.advance()
	switch char {
	case ' ', '\r', '\t':
		// ignore whitespace
		return nil
	case '\n':
		l.line++

		return nil
	case '"':
		return l.string()
	default:
		if k, ok := getReservedSymbol(char); ok {
			l.addToken(k, nil)

			return nil
		}
		if isDigit(char) {
			return l.integer()
		}
		if isAlpha(char) {
			return l.identifier()
		}
		if isSymbol(char) {
			return l.operator()
		}
	}

	return UnexpectedCharacterError{Line: l.line, Char: char}
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

	value := l.source[l.start+1 : l.current-1]
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

	value, err := strconv.Atoi(l.source[l.start:l.current])
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

	value := l.source[l.start:l.current]

	if k, ok := getKeyword(value); ok {
		l.addToken(k, nil)
	} else {
		l.addToken(token.IDENT, nil)
	}

	return nil
}

func getKeyword(str string) (token.Kind, bool) {
	keywords := map[string]token.Kind{
		"->":     token.ARROW,
		"<-":     token.BACKARROW,
		"|":      token.BAR,
		"=":      token.EQUAL,
		"case":   token.CASE,
		"def":    token.DEF,
		"fn":     token.FN,
		"infix":  token.INFIX,
		"infixl": token.INFIXL,
		"infixr": token.INFIXR,
		"let":    token.LET,
		"prim":   token.PRIM,
		"type":   token.TYPE,
		"with":   token.WITH,
	}

	if k, ok := keywords[str]; ok {
		return k, true
	}

	return token.IDENT, false
}

func isSymbol(c rune) bool {
	_, isReserved := getReservedSymbol(c)

	return c != '_' && !isReserved && (unicode.IsSymbol(c) || unicode.IsPunct(c))
}

func getReservedSymbol(char rune) (token.Kind, bool) {
	// These characters are reserved symbols, but they are not included in operator.
	reservedSymbols := map[rune]token.Kind{
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
	if k, ok := reservedSymbols[char]; ok {
		return k, true
	}

	return token.OPERATOR, false
}

func (l *lexer) operator() error {
	for isSymbol(l.peek()) {
		l.advance()
	}

	value := l.source[l.start:l.current]
	if k, ok := getKeyword(value); ok {
		l.addToken(k, nil)
	} else {
		l.addToken(token.OPERATOR, nil)
	}

	return nil
}
