package lexer_test

import (
	"testing"

	"github.com/takoeight0821/anma/lexer"
	"github.com/takoeight0821/anma/token"
)

func TestLexer(t *testing.T) {
	testcases := []struct {
		input  string
		tokens []token.Token
	}{
		{
			input: "1",
			tokens: []token.Token{
				{Kind: token.INTEGER, Lexeme: "1", Line: 1, Literal: 1},
				{Kind: token.EOF, Lexeme: "", Line: 1, Literal: nil},
			},
		},
		{
			input: "1 + 2",
			tokens: []token.Token{
				{Kind: token.INTEGER, Lexeme: "1", Line: 1, Literal: 1},
				{Kind: token.OPERATOR, Lexeme: "+", Line: 1, Literal: nil},
				{Kind: token.INTEGER, Lexeme: "2", Line: 1, Literal: 2},
				{Kind: token.EOF, Lexeme: "", Line: 1, Literal: nil},
			},
		},
		{
			input: "1\n + 2",
			tokens: []token.Token{
				{Kind: token.INTEGER, Lexeme: "1", Line: 1, Literal: 1},
				{Kind: token.OPERATOR, Lexeme: "+", Line: 2, Literal: nil},
				{Kind: token.INTEGER, Lexeme: "2", Line: 2, Literal: 2},
				{Kind: token.EOF, Lexeme: "", Line: 2, Literal: nil},
			},
		},
	}

	for _, testcase := range testcases {
		tokens, err := lexer.Lex(testcase.input)
		if err != nil {
			t.Errorf("Lex(%q) returned error: %v", testcase.input, err)
		}

		if len(tokens) != len(testcase.tokens) {
			t.Errorf("Lex(%q) returned %v, expected %v", testcase.input, tokens, testcase.tokens)
		}

		for i, token := range tokens {
			if token != testcase.tokens[i] {
				t.Errorf("Lex(%q) returned %v, expected %v", testcase.input, tokens, testcase.tokens)
				break
			}
		}
	}
}
