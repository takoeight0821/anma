package main

import "fmt"

type Parser struct {
	tokens  []Token
	current int
	err     error
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens, 0, nil}
}

func (p *Parser) Parse() (Node, error) {
	err := p.err
	return nil, err
}

// AST

type Node interface {
	fmt.Stringer
}
