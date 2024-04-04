package lexer_test

import (
	"os"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/takoeight0821/anma/lexer"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

func TestLexer(t *testing.T) {
	t.Parallel()
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
		{
			input: "ああ +いい",
			tokens: []token.Token{
				{Kind: token.IDENT, Lexeme: "ああ", Line: 1, Literal: nil},
				{Kind: token.OPERATOR, Lexeme: "+", Line: 1, Literal: nil},
				{Kind: token.IDENT, Lexeme: "いい", Line: 1, Literal: nil},
				{Kind: token.EOF, Lexeme: "", Line: 1, Literal: nil},
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

func TestGolden(t *testing.T) {
	t.Parallel()

	testfiles, err := utils.FindSourceFiles("../testdata")
	if err != nil {
		t.Errorf("failed to find test files: %v", err)
		return
	}

	for _, testfile := range testfiles {
		source, err := os.ReadFile(testfile)
		if err != nil {
			t.Errorf("failed to read %s: %v", testfile, err)
			return
		}

		tokens, err := lexer.Lex(string(source))
		if err != nil {
			t.Errorf("%s returned error: %v", testfile, err)
			return
		}

		var builder strings.Builder
		for _, token := range tokens {
			builder.WriteString(token.String())
			builder.WriteString("\n")
		}

		g := goldie.New(t)
		g.Assert(t, testfile, []byte(builder.String()))
	}
}
